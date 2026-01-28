package handlers

import (
	"encoding/json"
	"log"

	"github.com/freetorrent/freetorrent/internal/config"
	"github.com/freetorrent/freetorrent/internal/database"
	"github.com/freetorrent/freetorrent/internal/middleware"
	"github.com/freetorrent/freetorrent/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/stripe/stripe-go/v76"
	portalsession "github.com/stripe/stripe-go/v76/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/customer"
	"github.com/stripe/stripe-go/v76/webhook"
)

// Stripe Price IDs (set these in production via environment variables)
var stripePriceIDs = map[string]string{
	"starter":   "price_starter_monthly",   // Replace with actual Stripe price ID
	"pro":       "price_pro_monthly",       // Replace with actual Stripe price ID
	"unlimited": "price_unlimited_monthly", // Replace with actual Stripe price ID
}

type BillingHandler struct {
	db  *database.Database
	cfg *config.Config
}

func NewBillingHandler(db *database.Database, cfg *config.Config) *BillingHandler {
	if cfg.StripeSecretKey != "" {
		stripe.Key = cfg.StripeSecretKey
	}
	return &BillingHandler{
		db:  db,
		cfg: cfg,
	}
}

// GetSubscription returns the current user's subscription
func (h *BillingHandler) GetSubscription(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{
			Error: "invalid user",
		})
	}

	sub, err := h.db.GetSubscription(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to get subscription",
		})
	}

	// Handle nil subscription
	if sub == nil {
		return c.JSON(fiber.Map{
			"subscription": nil,
			"usage": models.UsageStats{
				UsedGB:          0,
				LimitGB:         2,
				ActiveTorrents:  0,
				ConcurrentLimit: 1,
				Plan:            "free",
			},
			"plans": models.Plans,
		})
	}

	// Get usage stats
	monthlyUsage, _ := h.db.GetMonthlyUsage(c.Context(), userID)
	activeTorrents, _ := h.db.CountActiveTorrents(c.Context(), userID)

	return c.JSON(fiber.Map{
		"subscription": sub,
		"usage": models.UsageStats{
			UsedGB:          float64(monthlyUsage) / (1024 * 1024 * 1024),
			LimitGB:         sub.DownloadLimitGB,
			ActiveTorrents:  activeTorrents,
			ConcurrentLimit: sub.ConcurrentLimit,
			Plan:            sub.Plan,
		},
		"plans": models.Plans,
	})
}

// CreateCheckoutSession creates a Stripe checkout session for subscription
func (h *BillingHandler) CreateCheckoutSession(c *fiber.Ctx) error {
	if h.cfg.StripeSecretKey == "" {
		return c.Status(fiber.StatusServiceUnavailable).JSON(models.ErrorResponse{
			Error: "billing not configured",
		})
	}

	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{
			Error: "invalid user",
		})
	}

	type CheckoutRequest struct {
		Plan       string `json:"plan"`
		SuccessURL string `json:"success_url"`
		CancelURL  string `json:"cancel_url"`
	}

	var req CheckoutRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "invalid request",
		})
	}

	// Validate plan
	priceID, ok := stripePriceIDs[req.Plan]
	if !ok || req.Plan == "free" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "invalid plan",
		})
	}

	// Get user
	user, err := h.db.GetUserByID(c.Context(), userID)
	if err != nil || user == nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{
			Error: "user not found",
		})
	}

	// Get or create Stripe customer
	var customerID string
	if user.StripeCustomerID != nil {
		customerID = *user.StripeCustomerID
	} else {
		// Create new Stripe customer
		customerParams := &stripe.CustomerParams{
			Email: stripe.String(user.Email),
			Metadata: map[string]string{
				"user_id": userID.String(),
			},
		}
		cust, err := customer.New(customerParams)
		if err != nil {
			log.Printf("Failed to create Stripe customer: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
				Error: "failed to create customer",
			})
		}
		customerID = cust.ID
		// TODO: Save customer ID to database
	}

	// Create checkout session
	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(customerID),
		Mode:     stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(req.SuccessURL),
		CancelURL:  stripe.String(req.CancelURL),
		Metadata: map[string]string{
			"user_id": userID.String(),
			"plan":    req.Plan,
		},
	}

	sess, err := checkoutsession.New(params)
	if err != nil {
		log.Printf("Failed to create checkout session: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to create checkout session",
		})
	}

	return c.JSON(fiber.Map{
		"checkout_url": sess.URL,
		"session_id":   sess.ID,
	})
}

// CreatePortalSession creates a Stripe billing portal session
func (h *BillingHandler) CreatePortalSession(c *fiber.Ctx) error {
	if h.cfg.StripeSecretKey == "" {
		return c.Status(fiber.StatusServiceUnavailable).JSON(models.ErrorResponse{
			Error: "billing not configured",
		})
	}

	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{
			Error: "invalid user",
		})
	}

	user, err := h.db.GetUserByID(c.Context(), userID)
	if err != nil || user == nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{
			Error: "user not found",
		})
	}

	if user.StripeCustomerID == nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "no billing account found",
		})
	}

	type PortalRequest struct {
		ReturnURL string `json:"return_url"`
	}

	var req PortalRequest
	if err := c.BodyParser(&req); err != nil {
		req.ReturnURL = "/"
	}

	params := &stripe.BillingPortalSessionParams{
		Customer:  user.StripeCustomerID,
		ReturnURL: stripe.String(req.ReturnURL),
	}

	sess, err := portalsession.New(params)
	if err != nil {
		log.Printf("Failed to create portal session: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to create portal session",
		})
	}

	return c.JSON(fiber.Map{
		"portal_url": sess.URL,
	})
}

// HandleWebhook processes Stripe webhook events
func (h *BillingHandler) HandleWebhook(c *fiber.Ctx) error {
	if h.cfg.StripeWebhookKey == "" {
		return c.Status(fiber.StatusServiceUnavailable).JSON(models.ErrorResponse{
			Error: "webhooks not configured",
		})
	}

	payload := c.Body()
	sigHeader := c.Get("Stripe-Signature")

	event, err := webhook.ConstructEvent(payload, sigHeader, h.cfg.StripeWebhookKey)
	if err != nil {
		log.Printf("Webhook signature verification failed: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "invalid signature",
		})
	}

	switch event.Type {
	case "checkout.session.completed":
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
				Error: "invalid event data",
			})
		}
		h.handleCheckoutCompleted(&sess)

	case "customer.subscription.created", "customer.subscription.updated":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
				Error: "invalid event data",
			})
		}
		h.handleSubscriptionUpdated(&sub)

	case "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
				Error: "invalid event data",
			})
		}
		h.handleSubscriptionCanceled(&sub)

	case "invoice.payment_failed":
		var inv stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
				Error: "invalid event data",
			})
		}
		h.handlePaymentFailed(&inv)
	}

	return c.JSON(fiber.Map{"received": true})
}

func (h *BillingHandler) handleCheckoutCompleted(sess *stripe.CheckoutSession) {
	log.Printf("Checkout completed for customer %s", sess.Customer.ID)
	// The subscription webhook will handle the actual update
}

func (h *BillingHandler) handleSubscriptionUpdated(sub *stripe.Subscription) {
	log.Printf("Subscription updated: %s, status: %s", sub.ID, sub.Status)

	// Determine plan from price ID
	plan := "free"
	if len(sub.Items.Data) > 0 {
		priceID := sub.Items.Data[0].Price.ID
		for p, id := range stripePriceIDs {
			if id == priceID {
				plan = p
				break
			}
		}
	}

	// Map Stripe status to our status
	status := "active"
	switch sub.Status {
	case stripe.SubscriptionStatusActive:
		status = "active"
	case stripe.SubscriptionStatusPastDue:
		status = "past_due"
	case stripe.SubscriptionStatusCanceled:
		status = "canceled"
	case stripe.SubscriptionStatusTrialing:
		status = "trialing"
	}

	log.Printf("Plan: %s, Status: %s", plan, status)
	
	// TODO: Find user by Stripe customer ID and update subscription
	// This requires adding h.db.GetUserByStripeCustomerID() method
	// Then call h.db.UpdateSubscription(ctx, userID, plan, status, models.Plans[plan])
}

func (h *BillingHandler) handleSubscriptionCanceled(sub *stripe.Subscription) {
	log.Printf("Subscription canceled: %s", sub.ID)
	// TODO: Downgrade user to free plan
}

func (h *BillingHandler) handlePaymentFailed(inv *stripe.Invoice) {
	log.Printf("Payment failed for customer %s", inv.Customer.ID)
	// TODO: Send notification to user, maybe restrict access
}
