import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { login, registerUser } from "@/api/jobs";

export function LoginPage() {
  const [isRegister, setIsRegister] = useState(false);
  const [email, setEmail] = useState("test@test.com");
  const [password, setPassword] = useState("password123");
  const [name, setName] = useState("");
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const navigate = useNavigate();

  const handleSubmit = async () => {
    setError("");
    setSuccess("");
    try {
      if (isRegister) {
        await registerUser(email, password, name);
        setSuccess("Registered successfully. Please sign in.");
        setIsRegister(false);
      } else {
        const token = await login(email, password);
        localStorage.setItem("token", token);
        navigate("/jobs");
      }
    } catch {
      setError(isRegister ? "Registration failed" : "Invalid credentials");
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <Card className="w-96">
        <CardHeader>
          <CardTitle>
            {isRegister ? "Create Account" : "Mini Databricks"}
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          {isRegister && (
            <input
              className="border rounded px-3 py-2 text-sm"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Full name"
            />
          )}
          <input
            className="border rounded px-3 py-2 text-sm"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="Email"
          />
          <input
            className="border rounded px-3 py-2 text-sm"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Password"
          />
          {error && <p className="text-red-500 text-sm">{error}</p>}
          {success && <p className="text-green-500 text-sm">{success}</p>}
          <Button onClick={handleSubmit}>
            {isRegister ? "Register" : "Sign In"}
          </Button>
          <button
            className="text-sm text-gray-500 hover:text-gray-700"
            onClick={() => {
              setIsRegister(!isRegister);
              setError("");
              setSuccess("");
            }}
          >
            {isRegister
              ? "Already have an account? Sign in"
              : "Don't have an account? Register"}
          </button>
        </CardContent>
      </Card>
    </div>
  );
}
