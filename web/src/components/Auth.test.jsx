import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import Auth from './Auth'

const messages = {
  appTagline: 'Deck analyzer',
  signOut: 'Sign out',
  signIn: 'Sign in',
  createAccount: 'Create account',
  noAccount: 'No account?',
  signUpFree: 'Sign up free',
  haveAccount: 'Have an account?',
  loading: 'Loading…',
  name: 'Name',
  yourName: 'Your name',
  email: 'Email',
  password: 'Password',
  confirmPassword: 'Confirm password',
  showPassword: 'Show',
  hidePassword: 'Hide',
  passwordStrengthLabel: 'Password strength',
  passwordStrengthWeak: 'Weak',
  passwordStrengthMedium: 'Medium',
  passwordStrengthStrong: 'Strong',
  passwordStrengthLength: 'At least 8 characters',
  passwordStrengthLowercase: 'One lowercase letter',
  passwordStrengthUppercase: 'One uppercase letter',
  passwordStrengthNumber: 'One number',
  passwordStrengthSymbol: 'One symbol',
  passwordStrengthHint: 'Use a mix of uppercase and lowercase letters, numbers and symbols.',
  forgotPasswordTitle: 'Recover password',
  forgotPasswordCta: 'Forgot password?',
  sendResetLink: 'Send reset link',
  resetLinkSent: 'If the email exists, you will receive a password reset link.',
  resetPasswordTitle: 'Set new password',
  resetPasswordAction: 'Update password',
  resetPasswordSuccess: 'Password updated. You can now sign in with your new credentials.',
  backToSignIn: 'Back to sign in',
  passwordMismatch: 'Passwords do not match',
  resetTokenMissing: 'Reset token is missing or invalid',
  proActivationFromPlansNote: 'Plan note',
}

function renderAuth() {
  return render(
    <Auth
      onLogin={vi.fn()}
      locale="en"
      messages={messages}
      onLocaleChange={vi.fn()}
    />,
  )
}

describe('Auth reset flow', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    window.history.replaceState({}, '', '/')
  })

  it('requests reset link from forgot-password mode', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      headers: { get: () => 'application/json' },
      json: vi.fn().mockResolvedValue({
        status: 'accepted',
        message: 'If the email exists, you will receive a password reset link.',
      }),
    }))

    renderAuth()

    fireEvent.click(screen.getByRole('button', { name: 'Forgot password?' }))
    fireEvent.change(screen.getByPlaceholderText('you@example.com'), { target: { value: 'user@example.com' } })
    fireEvent.click(screen.getByRole('button', { name: 'Send reset link' }))

    await waitFor(() => {
      expect(screen.getByText('If the email exists, you will receive a password reset link.')).toBeInTheDocument()
    })

    expect(fetch).toHaveBeenCalledWith('/api/v1/auth/forgot-password', expect.objectContaining({
      method: 'POST',
    }))
  })

  it('uses generic reset success message when forgot-password response is non-json', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      headers: { get: () => 'text/plain' },
      text: vi.fn().mockResolvedValue('accepted'),
    }))

    renderAuth()

    fireEvent.click(screen.getByRole('button', { name: 'Forgot password?' }))
    fireEvent.change(screen.getByPlaceholderText('you@example.com'), { target: { value: 'user@example.com' } })
    fireEvent.click(screen.getByRole('button', { name: 'Send reset link' }))

    await waitFor(() => {
      expect(screen.getByText('If the email exists, you will receive a password reset link.')).toBeInTheDocument()
    })
  })

  it('shows password complexity guidance in register mode', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      headers: { get: () => 'application/json' },
      json: vi.fn().mockResolvedValue({ token: 'abc', user: { id: 'u1' } }),
    })
    vi.stubGlobal('fetch', fetchMock)

    const onLogin = vi.fn()
    render(
      <Auth
        onLogin={onLogin}
        locale="en"
        messages={messages}
        onLocaleChange={vi.fn()}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: 'Sign up free' }))

    expect(screen.getAllByPlaceholderText('••••••••')).toHaveLength(2)

    fireEvent.change(screen.getAllByPlaceholderText('••••••••')[0], { target: { value: 'abc' } })
    fireEvent.change(screen.getAllByPlaceholderText('••••••••')[1], { target: { value: 'abc' } })

    expect(screen.getByText('Password strength')).toBeInTheDocument()
    expect(screen.getByText('Weak')).toBeInTheDocument()
    expect(screen.getByText('At least 8 characters')).toBeInTheDocument()

    fireEvent.change(screen.getAllByPlaceholderText('••••••••')[0], { target: { value: 'Abcdef12!' } })
    fireEvent.change(screen.getAllByPlaceholderText('••••••••')[1], { target: { value: 'Abcdef12!' } })
    expect(screen.getByText('Strong')).toBeInTheDocument()

    fireEvent.change(screen.getByPlaceholderText('you@example.com'), { target: { value: 'user@example.com' } })
    fireEvent.change(screen.getByPlaceholderText('Your name'), { target: { value: 'User Name' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create account' }))

    await waitFor(() => {
      expect(onLogin).toHaveBeenCalledWith('abc', { id: 'u1' })
    })

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/auth/register', expect.objectContaining({
      method: 'POST',
    }))
  })

  it('shows mismatch validation before register call', async () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)

    renderAuth()

    fireEvent.click(screen.getByRole('button', { name: 'Sign up free' }))
    fireEvent.change(screen.getByPlaceholderText('Your name'), { target: { value: 'User Name' } })
    fireEvent.change(screen.getByPlaceholderText('you@example.com'), { target: { value: 'user@example.com' } })
    fireEvent.change(screen.getAllByPlaceholderText('••••••••')[0], { target: { value: 'NewPass123!' } })
    fireEvent.change(screen.getAllByPlaceholderText('••••••••')[1], { target: { value: 'Different123!' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create account' }))

    await waitFor(() => {
      expect(screen.getByText('Passwords do not match')).toBeInTheDocument()
    })

    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('shows fallback error when register response is non-json', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      headers: { get: () => 'text/plain' },
      text: vi.fn().mockResolvedValue('ok'),
    })
    vi.stubGlobal('fetch', fetchMock)

    renderAuth()

    fireEvent.click(screen.getByRole('button', { name: 'Sign up free' }))
    fireEvent.change(screen.getByPlaceholderText('Your name'), { target: { value: 'User Name' } })
    fireEvent.change(screen.getByPlaceholderText('you@example.com'), { target: { value: 'user@example.com' } })
    fireEvent.change(screen.getAllByPlaceholderText('••••••••')[0], { target: { value: 'NewPass123!' } })
    fireEvent.change(screen.getAllByPlaceholderText('••••••••')[1], { target: { value: 'NewPass123!' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create account' }))

    await waitFor(() => {
      expect(screen.getByText('Request failed')).toBeInTheDocument()
    })
  })

  it('submits reset token and new password from query param', async () => {
    window.history.replaceState({}, '', '/reset-password?token=tok-abc')

    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      headers: { get: () => 'application/json' },
      json: vi.fn().mockResolvedValue({ status: 'ok' }),
    }))

    renderAuth()

    fireEvent.change(screen.getAllByPlaceholderText('••••••••')[0], { target: { value: 'NewPass123' } })
    fireEvent.change(screen.getAllByPlaceholderText('••••••••')[1], { target: { value: 'NewPass123' } })
    fireEvent.click(screen.getByRole('button', { name: 'Update password' }))

    await waitFor(() => {
      expect(screen.getByText('Password updated. You can now sign in with your new credentials.')).toBeInTheDocument()
    })

    expect(fetch).toHaveBeenCalledWith('/api/v1/auth/reset-password', expect.objectContaining({
      method: 'POST',
    }))
  })

  it('shows mismatch validation before reset call', async () => {
    window.history.replaceState({}, '', '/reset-password?token=tok-mismatch')
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)

    renderAuth()

    fireEvent.change(screen.getAllByPlaceholderText('••••••••')[0], { target: { value: 'NewPass123' } })
    fireEvent.change(screen.getAllByPlaceholderText('••••••••')[1], { target: { value: 'Different123' } })
    fireEvent.click(screen.getByRole('button', { name: 'Update password' }))

    await waitFor(() => {
      expect(screen.getByText('Passwords do not match')).toBeInTheDocument()
    })

    expect(fetchMock).not.toHaveBeenCalled()
  })
})
