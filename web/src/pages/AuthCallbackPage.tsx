import { useEffect, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { setToken } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

// Página onde a callback do Google aterrissa.
//
// O backend redireciona para cá com o resultado no FRAGMENTO da URL
// (#token=...&expiresAt=... ou #error=...). Fragmento não é enviado ao
// servidor, então o token não aparece em log de acesso nem em Referer —
// foi por isso que escolhemos fragmento em vez de query string.
export function AuthCallbackPage() {
  const navigate = useNavigate();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    // location.hash vem com o "#" na frente; o resto é um par chave=valor
    // igual a uma query string.
    const params = new URLSearchParams(window.location.hash.slice(1));
    const token = params.get("token");
    const failure = params.get("error");

    if (token) {
      setToken(token);
      // Limpa o fragmento para o token não ficar na barra de endereços
      // nem no histórico.
      window.history.replaceState(null, "", window.location.pathname);
      navigate({ to: "/forms", replace: true });
      return;
    }
    setError(failure ?? "não recebemos o token do servidor");
  }, [navigate]);

  if (!error) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background px-4">
        <p className="text-sm text-muted-foreground">Entrando...</p>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle>Não foi possível entrar</CardTitle>
          <CardDescription>O login com Google não foi concluído.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-destructive">{error}</p>
          <Button className="w-full" onClick={() => navigate({ to: "/login", replace: true })}>
            Voltar para o login
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
