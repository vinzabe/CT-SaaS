import { Link } from 'react-router-dom'
import { 
  Cloud, 
  Shield, 
  Zap, 
  Globe, 
  Lock,
  Check,
  ArrowRight
} from 'lucide-react'

export function LandingPage() {
  const features = [
    {
      icon: Zap,
      title: 'Lightning Fast',
      description: 'Download at maximum speed with our optimized torrent engine and global infrastructure.',
    },
    {
      icon: Shield,
      title: 'Post-Quantum Security',
      description: 'Future-proof encryption using NIST-approved post-quantum cryptographic algorithms.',
    },
    {
      icon: Globe,
      title: 'Access Anywhere',
      description: 'Convert any torrent or magnet link to a direct download link accessible from any device.',
    },
    {
      icon: Lock,
      title: 'Privacy First',
      description: 'Your downloads are private. We never share or log your activity.',
    },
  ]

  const plans = [
    {
      name: 'Free',
      price: 0,
      features: ['2 GB/month', '1 concurrent download', '24h file retention', 'Basic support'],
      cta: 'Get Started',
      popular: false,
    },
    {
      name: 'Starter',
      price: 5,
      features: ['50 GB/month', '3 concurrent downloads', '7 days file retention', 'Email support'],
      cta: 'Start Free Trial',
      popular: false,
    },
    {
      name: 'Pro',
      price: 15,
      features: ['500 GB/month', '10 concurrent downloads', '30 days file retention', 'Priority support', 'API access'],
      cta: 'Start Free Trial',
      popular: true,
    },
    {
      name: 'Unlimited',
      price: 30,
      features: ['Unlimited bandwidth', '25 concurrent downloads', '90 days file retention', '24/7 support', 'Full API access'],
      cta: 'Start Free Trial',
      popular: false,
    },
  ]

  return (
    <div className="min-h-screen bg-white">
      {/* Navigation */}
      <nav className="fixed top-0 left-0 right-0 z-50 bg-white/80 backdrop-blur-lg border-b border-gray-100">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <Link to="/" className="flex items-center gap-2">
              <Cloud className="w-8 h-8 text-primary-600" />
              <span className="text-xl font-bold text-gray-900">Grant's Torrent</span>
            </Link>
            <div className="flex items-center gap-4">
              <Link to="/login" className="text-gray-600 hover:text-gray-900 font-medium">
                Sign in
              </Link>
              <Link to="/register" className="btn-primary">
                Get Started
              </Link>
            </div>
          </div>
        </div>
      </nav>

      {/* Hero Section */}
      <section className="pt-32 pb-20 px-4 sm:px-6 lg:px-8">
        <div className="max-w-7xl mx-auto text-center">
          <div className="inline-flex items-center gap-2 bg-primary-50 text-primary-700 px-4 py-2 rounded-full text-sm font-medium mb-6">
            <Shield className="w-4 h-4" />
            Post-Quantum Secure
          </div>
          <h1 className="text-5xl sm:text-6xl lg:text-7xl font-bold text-gray-900 mb-6">
            Torrent to Direct
            <br />
            <span className="text-primary-600">Download</span>
          </h1>
          <p className="text-xl text-gray-600 max-w-2xl mx-auto mb-10">
            Convert any torrent or magnet link to a direct HTTP download. 
            Fast, secure, and accessible from any device.
          </p>
          <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
            <Link to="/register" className="btn-primary text-lg px-8 py-4">
              Start for Free
              <ArrowRight className="w-5 h-5 ml-2" />
            </Link>
            <Link to="/login" className="btn-secondary text-lg px-8 py-4">
              Sign in
            </Link>
          </div>
          <div className="mt-6 p-4 bg-gray-100 rounded-lg inline-block">
            <p className="text-sm text-gray-600">
              <span className="font-medium">Try the demo:</span>{' '}
              <code className="bg-gray-200 px-2 py-0.5 rounded">demo@grants.torrent</code> / <code className="bg-gray-200 px-2 py-0.5 rounded">demo123</code>
            </p>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section className="py-20 bg-gray-50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-16">
            <h2 className="text-3xl sm:text-4xl font-bold text-gray-900 mb-4">
              Why Choose Grant's Torrent?
            </h2>
            <p className="text-lg text-gray-600 max-w-2xl mx-auto">
              Built with modern technology for speed, security, and reliability.
            </p>
          </div>
          <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-8">
            {features.map((feature, index) => (
              <div key={index} className="bg-white rounded-2xl p-6 shadow-sm border border-gray-100">
                <div className="w-12 h-12 bg-primary-100 rounded-xl flex items-center justify-center mb-4">
                  <feature.icon className="w-6 h-6 text-primary-600" />
                </div>
                <h3 className="text-lg font-semibold text-gray-900 mb-2">
                  {feature.title}
                </h3>
                <p className="text-gray-600">
                  {feature.description}
                </p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* How It Works */}
      <section className="py-20">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-16">
            <h2 className="text-3xl sm:text-4xl font-bold text-gray-900 mb-4">
              How It Works
            </h2>
          </div>
          <div className="grid md:grid-cols-3 gap-8">
            {[
              { step: '1', title: 'Paste Link', description: 'Paste your magnet link or upload a .torrent file' },
              { step: '2', title: 'We Download', description: 'Our servers download the torrent at high speed' },
              { step: '3', title: 'Direct Link', description: 'Get a direct HTTP link to download from anywhere' },
            ].map((item) => (
              <div key={item.step} className="text-center">
                <div className="w-16 h-16 bg-primary-600 text-white rounded-2xl flex items-center justify-center text-2xl font-bold mx-auto mb-4">
                  {item.step}
                </div>
                <h3 className="text-xl font-semibold text-gray-900 mb-2">
                  {item.title}
                </h3>
                <p className="text-gray-600">
                  {item.description}
                </p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Pricing Section */}
      <section className="py-20 bg-gray-50">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-16">
            <h2 className="text-3xl sm:text-4xl font-bold text-gray-900 mb-4">
              Simple, Transparent Pricing
            </h2>
            <p className="text-lg text-gray-600">
              Start free, upgrade when you need more
            </p>
          </div>
          <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-8">
            {plans.map((plan) => (
              <div
                key={plan.name}
                className={`bg-white rounded-2xl p-6 ${
                  plan.popular 
                    ? 'ring-2 ring-primary-600 shadow-lg scale-105' 
                    : 'border border-gray-200'
                }`}
              >
                {plan.popular && (
                  <div className="bg-primary-600 text-white text-xs font-medium px-3 py-1 rounded-full inline-block mb-4">
                    Most Popular
                  </div>
                )}
                <h3 className="text-xl font-bold text-gray-900 mb-2">{plan.name}</h3>
                <div className="mb-6">
                  <span className="text-4xl font-bold text-gray-900">${plan.price}</span>
                  <span className="text-gray-500">/month</span>
                </div>
                <ul className="space-y-3 mb-6">
                  {plan.features.map((feature, i) => (
                    <li key={i} className="flex items-center gap-2 text-sm text-gray-600">
                      <Check className="w-5 h-5 text-green-500 flex-shrink-0" />
                      {feature}
                    </li>
                  ))}
                </ul>
                <Link
                  to="/register"
                  className={`block text-center py-3 rounded-lg font-medium transition-colors ${
                    plan.popular
                      ? 'bg-primary-600 text-white hover:bg-primary-700'
                      : 'bg-gray-100 text-gray-900 hover:bg-gray-200'
                  }`}
                >
                  {plan.cta}
                </Link>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-20">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
          <h2 className="text-3xl sm:text-4xl font-bold text-gray-900 mb-4">
            Ready to Get Started?
          </h2>
          <p className="text-lg text-gray-600 mb-8">
            Create your free account and start downloading in seconds.
          </p>
          <Link to="/register" className="btn-primary text-lg px-8 py-4">
            Create Free Account
            <ArrowRight className="w-5 h-5 ml-2" />
          </Link>
        </div>
      </section>

      {/* Footer */}
      <footer className="bg-gray-900 text-gray-400 py-12">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex flex-col md:flex-row items-center justify-between gap-4">
            <div className="flex items-center gap-2">
              <Cloud className="w-6 h-6 text-primary-500" />
              <span className="text-white font-semibold">Grant's Torrent</span>
            </div>
            <p className="text-sm">
              &copy; {new Date().getFullYear()} Grant's Torrent. All rights reserved.
            </p>
          </div>
        </div>
      </footer>
    </div>
  )
}
