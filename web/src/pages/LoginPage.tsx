import { useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useMutation, useQuery } from "@tanstack/react-query";
import { api, apiBaseUrl, errorMessage, setToken } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

// Uma tela só para login e registro: o formulário é idêntico, muda a rota.
export function LoginPage() {
  const navigate = useNavigate();
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  // O botão do Google só aparece se o backend tiver as credenciais: sem
  // elas /auth/google responderia 503 e o botão seria um beco sem saída.
  const config = useQuery({
    queryKey: ["config"],
    queryFn: async () => {
      const { data, error } = await api.GET("/config");
      if (error) throw error;
      return data;
    },
  });

  const mutation = useMutation({
    mutationFn: async () => {
      // As duas chamadas são escritas separadamente para o openapi-fetch
      // resolver os tipos de cada operação.
      const body = { email, password };
      const { data, error } =
        mode === "login"
          ? await api.POST("/auth/login", { body })
          : await api.POST("/auth/register", { body });
      if (error) throw error;
      return data;
    },
    onSuccess: (data) => {
      setToken(data.token);
      navigate({ to: "/forms" });
    },
  });

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle>{mode === "login" ? "Entrar" : "Criar conta"}</CardTitle>
          <CardDescription>Área administrativa do Form Builder</CardDescription>
        </CardHeader>
        <CardContent>
          <form
            className="space-y-4"
            onSubmit={(e) => {
              e.preventDefault();
              mutation.mutate();
            }}
          >
            <div className="space-y-1.5">
              <Label htmlFor="email">E-mail</Label>
              <Input
                id="email"
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="admin@example.com"
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="password">Senha</Label>
              <Input
                id="password"
                type="password"
                required
                minLength={8}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="mínimo 8 caracteres"
              />
            </div>

            {mutation.isError && (
              <p className="text-sm text-destructive">{errorMessage(mutation.error)}</p>
            )}

            <Button type="submit" className="w-full" disabled={mutation.isPending}>
              {mutation.isPending ? "Enviando..." : mode === "login" ? "Entrar" : "Criar conta"}
            </Button>
          </form>

          {config.data?.googleAuth && (
            <>
              {/* Separador entre as duas formas de entrar. */}
              <div className="my-4 flex items-center gap-3">
                <span className="h-px flex-1 bg-border" />
                <span className="text-xs text-muted-foreground">ou</span>
                <span className="h-px flex-1 bg-border" />
              </div>

              {/* Navegação de verdade, não fetch: /auth/google responde um
                  redirect para a tela de consentimento do Google. */}
              <Button variant="outline" className="w-full" asChild>
                <a href={`${apiBaseUrl}/auth/google`}>Entrar com Google</a>
              </Button>
            </>
          )}

          <Button
            variant="link"
            className="mt-2 w-full"
            onClick={() => {
              setMode(mode === "login" ? "register" : "login");
              mutation.reset();
            }}
          >
            {mode === "login" ? "Não tenho conta — registrar" : "Já tenho conta — entrar"}
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
