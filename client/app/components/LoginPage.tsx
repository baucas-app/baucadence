import { login } from "api/api";
import { useState } from "react";
import { Eye, EyeOff, CircleAlert } from "lucide-react";
import "../login.css";

export default function LoginPage() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [remember, setRemember] = useState(true);
  const [showPassword, setShowPassword] = useState(false);

  const loginHandler = (e: React.FormEvent) => {
    e.preventDefault();
    if (!username || !password) {
      setError("username and password are required");
      return;
    }
    setError("");
    setLoading(true);
    login(username, password, remember)
      .then((r) => {
        if (r.status >= 200 && r.status < 300) {
          window.location.reload();
        } else {
          r.json().then((r) => setError(r.error));
        }
      })
      .catch((err) => setError(String(err)))
      .finally(() => setLoading(false));
  };

  return (
    <div className="login-page">
      <div className="login-stage">
        <div className="login-wordmark">
          <span className="login-mark">
            <svg viewBox="0 0 24 24" fill="none">
              <ellipse
                cx="7"
                cy="18.2"
                rx="3"
                ry="2.3"
                transform="rotate(-18 7 18.2)"
                fill="#f7f7f5"
              />
              <ellipse
                cx="17.3"
                cy="15.4"
                rx="3"
                ry="2.3"
                transform="rotate(-18 17.3 15.4)"
                fill="#f7f7f5"
              />
              <path
                d="M9.8 17.3V6.2l3.7-2.7 3.8 2.7v8.9"
                stroke="#f7f7f5"
                strokeWidth="1.7"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
          </span>
          <span>BauCadence</span>
        </div>

        <div className="login-card">
          <h1>Welcome back</h1>
          <p className="login-sub">
            Log in for <strong>admin tools</strong> — editing metadata,
            merging duplicates, and managing connected sources.
          </p>

          <form onSubmit={loginHandler} className="login-form">
            <div className="login-field">
              <label htmlFor="login-username">Username</label>
              <input
                id="login-username"
                name="koito-username"
                type="text"
                placeholder="admin"
                autoComplete="username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
              />
            </div>
            <div className="login-field login-field-pw">
              <label htmlFor="login-password">Password</label>
              <input
                id="login-password"
                name="koito-password"
                type={showPassword ? "text" : "password"}
                placeholder="••••••••••••"
                autoComplete="current-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
              <button
                type="button"
                className="login-pw-toggle"
                aria-label={showPassword ? "Hide password" : "Show password"}
                onClick={() => setShowPassword(!showPassword)}
              >
                {showPassword ? <EyeOff size={16} /> : <Eye size={16} />}
              </button>
            </div>

            <div className="login-row">
              <div className="login-remember">
                <input
                  type="checkbox"
                  id="login-remember"
                  checked={remember}
                  onChange={() => setRemember(!remember)}
                />
                <label htmlFor="login-remember">Remember me</label>
              </div>
            </div>

            {error && (
              <div className="login-error">
                <CircleAlert size={16} />
                <span>{error}</span>
              </div>
            )}

            <button type="submit" className="login-submit" disabled={loading}>
              {loading ? "Logging in…" : "Log in"}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}
