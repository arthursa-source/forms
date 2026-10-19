import { useEffect, useState } from "react";
import { useNavigate, useParams } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Trash2 } from "lucide-react";
import { api, errorMessage } from "@/lib/api";
import type { components } from "@/api/schema";
import { AppShell } from "@/components/AppShell";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Checkbox } from "@/components/ui/checkbox";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

// O tipo do campo vem da spec: se o backend adicionar um tipo, ele aparece
// aqui depois de regenerar o client.
type FieldBody = components["schemas"]["FieldBody"];
type FieldType = FieldBody["type"];

const FIELD_TYPES: { value: FieldType; label: string }[] = [
  { value: "text", label: "Texto" },
  { value: "number", label: "Número" },
  { value: "select", label: "Seleção" },
  { value: "checkbox", label: "Caixa de marcação" },
];

// Estado local de um campo em edição. As opções do select são editadas
// como texto separado por vírgula, e convertidas na hora de salvar.
type DraftField = {
  label: string;
  type: FieldType;
  required: boolean;
  optionsText: string;
};

const emptyField: DraftField = { label: "", type: "text", required: false, optionsText: "" };

export function FormEditorPage({ mode }: { mode: "new" | "edit" }) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const params = useParams({ strict: false }) as { formId?: string };
  const formId = params.formId ? Number(params.formId) : undefined;

  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [fields, setFields] = useState<DraftField[]>([{ ...emptyField }]);

  const existing = useQuery({
    queryKey: ["forms", formId],
    enabled: mode === "edit" && formId !== undefined,
    queryFn: async () => {
      const { data, error } = await api.GET("/forms/{formId}", {
        params: { path: { formId: formId! } },
      });
      if (error) throw error;
      return data;
    },
  });

  // Carrega os dados existentes no estado local uma vez que a query resolve.
  useEffect(() => {
    if (!existing.data) return;
    setTitle(existing.data.title);
    setDescription(existing.data.description);
    setFields(
      existing.data.fields.map((f) => ({
        label: f.label,
        type: f.type,
        required: f.required,
        optionsText: (f.options ?? []).join(", "),
      })),
    );
  }, [existing.data]);

  const save = useMutation({
    mutationFn: async () => {
      const body = {
        title,
        description,
        fields: fields.map<FieldBody>((f) => ({
          label: f.label,
          type: f.type,
          required: f.required,
          // options só vai quando é select; o backend recusa nos outros tipos.
          ...(f.type === "select"
            ? {
                options: f.optionsText
                  .split(",")
                  .map((o) => o.trim())
                  .filter(Boolean),
              }
            : {}),
        })),
      };

      if (mode === "edit" && formId !== undefined) {
        const { data, error } = await api.PUT("/forms/{formId}", {
          params: { path: { formId } },
          body,
        });
        if (error) throw error;
        return data;
      }
      const { data, error } = await api.POST("/forms", { body });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["forms"] });
      navigate({ to: "/forms" });
    },
  });

  function updateField(index: number, patch: Partial<DraftField>) {
    setFields((current) => current.map((f, i) => (i === index ? { ...f, ...patch } : f)));
  }

  return (
    <AppShell>
      <h1 className="mb-6 text-2xl font-semibold">
        {mode === "edit" ? "Editar formulário" : "Novo formulário"}
      </h1>

      {existing.isLoading && <p className="text-sm text-muted-foreground">Carregando...</p>}

      <form
        className="space-y-6"
        onSubmit={(e) => {
          e.preventDefault();
          save.mutate();
        }}
      >
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Dados do formulário</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="title">Título</Label>
              <Input id="title" required value={title} onChange={(e) => setTitle(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="description">Descrição</Label>
              <Textarea
                id="description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
              />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex-row items-center justify-between space-y-0">
            <CardTitle className="text-base">Campos</CardTitle>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setFields((f) => [...f, { ...emptyField }])}
            >
              Adicionar campo
            </Button>
          </CardHeader>
          <CardContent className="space-y-4">
            {fields.length === 0 && (
              <p className="text-sm text-muted-foreground">
                Um formulário precisa de ao menos um campo para ser publicado.
              </p>
            )}

            {fields.map((field, index) => (
              <div key={index} className="rounded-md border p-4">
                <div className="flex flex-wrap items-end gap-3">
                  <div className="min-w-[200px] flex-1 space-y-1.5">
                    <Label htmlFor={`label-${index}`}>Pergunta</Label>
                    <Input
                      id={`label-${index}`}
                      required
                      value={field.label}
                      onChange={(e) => updateField(index, { label: e.target.value })}
                      placeholder="Ex: Qual seu nome?"
                    />
                  </div>

                  <div className="w-44 space-y-1.5">
                    <Label>Tipo</Label>
                    <Select
                      value={field.type}
                      onValueChange={(value) => updateField(index, { type: value as FieldType })}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {FIELD_TYPES.map((t) => (
                          <SelectItem key={t.value} value={t.value}>
                            {t.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="flex h-9 items-center gap-2">
                    <Checkbox
                      id={`required-${index}`}
                      checked={field.required}
                      onCheckedChange={(checked) => updateField(index, { required: checked === true })}
                    />
                    <Label htmlFor={`required-${index}`}>Obrigatório</Label>
                  </div>

                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="text-destructive"
                    aria-label="Remover campo"
                    onClick={() => setFields((f) => f.filter((_, i) => i !== index))}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>

                {field.type === "select" && (
                  <div className="mt-3 space-y-1.5">
                    <Label htmlFor={`options-${index}`}>Opções (separadas por vírgula)</Label>
                    <Input
                      id={`options-${index}`}
                      required
                      value={field.optionsText}
                      onChange={(e) => updateField(index, { optionsText: e.target.value })}
                      placeholder="Google, Indicação, Redes sociais"
                    />
                  </div>
                )}
              </div>
            ))}
          </CardContent>
        </Card>

        {save.isError && <p className="text-sm text-destructive">{errorMessage(save.error)}</p>}

        <div className="flex gap-2">
          <Button type="submit" disabled={save.isPending}>
            {save.isPending ? "Salvando..." : "Salvar"}
          </Button>
          <Button type="button" variant="outline" onClick={() => navigate({ to: "/forms" })}>
            Cancelar
          </Button>
        </div>
      </form>
    </AppShell>
  );
}
