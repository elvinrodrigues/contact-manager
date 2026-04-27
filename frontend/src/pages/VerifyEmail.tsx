import { useEffect, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { Loader2, CheckCircle, XCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { authApi } from "@/services/api";

export default function VerifyEmail() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const token = searchParams.get("token");

  const [loading, setLoading] = useState(true);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!token) {
      setError("No verification token provided");
      setLoading(false);
      return;
    }

    authApi
      .verifyEmail(token)
      .then(() => {
        setSuccess(true);
      })
      .catch((err) => {
        setError(err.message || "Verification failed");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [token]);

  // Auto-redirect to login 3 seconds after successful verification
  useEffect(() => {
    if (success) {
      const timer = setTimeout(() => navigate("/login", { replace: true }), 3000);
      return () => clearTimeout(timer);
    }
  }, [success, navigate]);

  return (
    <div className="flex min-h-screen items-center justify-center bg-surface px-4">
      <div className="w-full max-w-sm text-center space-y-6">
        <h1 className="text-2xl font-bold tracking-tight">Email Verification</h1>

        {loading ? (
          <div className="flex flex-col items-center gap-4 text-muted-foreground">
            <Loader2 className="h-8 w-8 animate-spin text-primary" />
            <p>Verifying your email address...</p>
          </div>
        ) : success ? (
          <div className="flex flex-col items-center gap-4">
            <CheckCircle className="h-12 w-12 text-success" />
            <p className="text-muted-foreground">
              Your email has been successfully verified!
            </p>
            <p className="text-xs text-muted-foreground">
              Redirecting to login in 3 seconds…
            </p>
            <Button asChild className="w-full mt-4">
              <Link to="/login">Go to Login</Link>
            </Button>
          </div>
        ) : (
          <div className="flex flex-col items-center gap-4">
            <XCircle className="h-12 w-12 text-danger" />
            <p className="text-danger">{error}</p>
            <Button asChild variant="outline" className="w-full mt-4">
              <Link to="/login">Return to Login</Link>
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
