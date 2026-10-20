import { useState } from 'react'
import { login } from '../../../api/auth'
import type { AuthSession } from '../../../store/authStore'
import './LoginPage.css'

type Props = {
  onLogin: (session: AuthSession, remember: boolean) => void
}

const UserIcon = () => (
  <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
    <path d="M12 2a10 10 0 1 0 0 20 10 10 0 0 0 0-20Zm0 4a3.5 3.5 0 1 1 0 7 3.5 3.5 0 0 1 0-7Zm0 13.2a8 8 0 0 1-6-2.7c.04-2 4-3.1 6-3.1s5.96 1.1 6 3.1a8 8 0 0 1-6 2.7Z" />
  </svg>
)

const EyeIcon = ({ off }: { off: boolean }) => (
  <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
    <path d="M2 12s3.6-7 10-7 10 7 10 7-3.6 7-10 7S2 12 2 12Z" />
    <circle cx="12" cy="12" r="3" />
    {off && <path d="M3 3l18 18" />}
  </svg>
)

export default function LoginPage({ onLogin }: Props) {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [remember, setRemember] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  const submit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setError('')
    setSubmitting(true)
    try {
      const session = await login(email.trim(), password)
      onLogin(session, remember)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to sign in. Please try again.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <main className="lg-screen">
      {/* Left: welcome panel */}
      <section className="lg-hero">
        <div className="lg-hero-inner">
          <p className="lg-welcome">WELCOME TO</p>
          <div className="lg-logo">
            <span className="lg-logo-name">ErpMate</span>
            <span className="lg-logo-tag">ENTERPRISE RESOURCE PLANNING</span>
          </div>
          <p className="lg-hero-text">
            ErpMate brings procurement, production, inventory and sales into one
            system, to help bridge the gap on efficiency, profitability and
            customer satisfaction.
          </p>
        </div>
      </section>

      {/* Right: login form */}
      <section className="lg-panel" aria-labelledby="lg-title">
        <div className="lg-card">
          <h1 id="lg-title">Login</h1>
          <p className="lg-sub">
            Login with our ERP and enjoy a smooth, hassle-free environment
          </p>

          <form className="lg-form" onSubmit={submit}>
            <div className="lg-input">
              <input
                id="lg-email"
                type="email"
                autoComplete="username"
                placeholder="Username"
                aria-label="Username"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                required
                autoFocus
              />
              <span className="lg-input-icon"><UserIcon /></span>
            </div>

            <div className="lg-input">
              <input
                id="lg-password"
                type={showPassword ? 'text' : 'password'}
                autoComplete="current-password"
                placeholder="Password"
                aria-label="Password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                required
              />
              <button
                className="lg-input-icon lg-eye"
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                aria-label={showPassword ? 'Hide password' : 'Show password'}
              >
                <EyeIcon off={!showPassword} />
              </button>
            </div>

            <div className="lg-options">
              <label className="lg-remember">
                <input
                  type="checkbox"
                  checked={remember}
                  onChange={(event) => setRemember(event.target.checked)}
                />
                <span>Keep me signed in</span>
              </label>
              <button
                className="lg-link"
                type="button"
                onClick={() =>
                  setError('Please contact your system administrator to reset your password.')
                }
              >
                Forgot password?
              </button>
            </div>

            {error && (
              <div className="lg-error" role="alert">
                {error}
              </div>
            )}

            <button className="lg-submit" type="submit" disabled={submitting}>
              {submitting ? (
                <>
                  <span className="lg-spinner" aria-hidden="true" /> Signing in…
                </>
              ) : (
                'Login'
              )}
            </button>
          </form>
        </div>
      </section>
    </main>
  )
}