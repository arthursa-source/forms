"""Smoke test do backend. Roda contra http://localhost:8080."""
import json, urllib.request, urllib.error, sys

BASE = "http://localhost:8080"
ok = fail = 0

def call(method, path, body=None, token=None):
    req = urllib.request.Request(BASE + path, method=method)
    data = None
    if body is not None:
        data = json.dumps(body).encode()
        req.add_header("Content-Type", "application/json")
    if token:
        req.add_header("Authorization", "Bearer " + token)
    try:
        with urllib.request.urlopen(req, data) as r:
            return r.status, json.loads(r.read() or b"{}")
    except urllib.error.HTTPError as e:
        raw = e.read()
        try:
            return e.code, json.loads(raw or b"{}")
        except Exception:
            return e.code, {"raw": raw.decode(errors="replace")}

def check(name, got, want):
    global ok, fail
    if got == want:
        ok += 1; print(f"  ok   {name} ({got})")
    else:
        fail += 1; print(f"  FAIL {name}: esperado {want}, recebido {got}")

print("== auth ==")
st, _ = call("POST", "/auth/register", {"email": "smoke@example.com", "password": "senha12345"})
check("register (201 ou 409)", st in (201, 409), True)
st, s = call("POST", "/auth/login", {"email": "smoke@example.com", "password": "senha12345"})
check("login", st, 200)
tok = s["token"]

print("== criar formulário ==")
st, form = call("POST", "/forms", {
    "title": "Pesquisa Smoke",
    "description": "teste automatizado",
    "fields": [
        {"label": "Nome", "type": "text", "required": True},
        {"label": "Idade", "type": "number", "required": False},
        {"label": "Canal", "type": "select", "required": True,
         "options": ["Google", "Indicação", "Redes sociais"]},
        {"label": "Aceito os termos", "type": "checkbox", "required": True},
    ]}, token=tok)
check("createForm", st, 201)
fid = form["id"]
check("nasce como draft", form["status"], "draft")

st, pub = call("POST", f"/forms/{fid}/publish", token=tok)
check("publish", st, 200)
slug = pub["slug"]
check("publicPath", pub["publicPath"], "/f/" + slug)

print("== rota pública ==")
st, pf = call("GET", f"/public/forms/{slug}")
check("GET público sem token", st, 200)
ids = {f["label"]: f["id"] for f in pf["fields"]}
nome, idade, canal, termos = ids["Nome"], ids["Idade"], ids["Canal"], ids["Aceito os termos"]

print("== validação da submissão ==")
st, _ = call("POST", f"/public/forms/{slug}/responses",
             {"answers": {str(nome): "Maria Silva", str(idade): 34,
                          str(canal): "Indicação", str(termos): True}})
check("submissão válida", st, 201)

st, _ = call("POST", f"/public/forms/{slug}/responses",
             {"answers": {str(nome): "João", str(idade): "42",
                          str(canal): "Google", str(termos): True}})
check('number aceito como texto "42"', st, 201)

st, _ = call("POST", f"/public/forms/{slug}/responses",
             {"answers": {str(nome): "Ana", str(canal): "Google", str(termos): True}})
check("campo opcional ausente", st, 201)

st, e = call("POST", f"/public/forms/{slug}/responses",
             {"answers": {str(canal): "Google", str(termos): True}})
check("obrigatório faltando -> 422", st, 422)

st, e = call("POST", f"/public/forms/{slug}/responses",
             {"answers": {str(nome): "X", str(canal): "Facebook", str(termos): True}})
check("opção de select inválida -> 422", st, 422)

st, _ = call("POST", f"/public/forms/{slug}/responses",
             {"answers": {str(nome): "X", str(idade): "abc",
                          str(canal): "Google", str(termos): True}})
check("number inválido -> 422", st, 422)

st, _ = call("POST", f"/public/forms/{slug}/responses",
             {"answers": {str(nome): "X", str(canal): "Google", str(termos): False}})
check("checkbox obrigatório desmarcado -> 422", st, 422)

st, _ = call("POST", f"/public/forms/{slug}/responses",
             {"answers": {str(nome): "X", str(canal): "Google",
                          str(termos): True, "999999": "intruso"}})
check("chave desconhecida -> 422", st, 422)

print("== listagem de respostas ==")
st, lst = call("GET", f"/forms/{fid}/responses", token=tok)
check("listResponses", st, 200)
check("total de respostas", lst["total"], 3)
print("        " + " | ".join(f"{a['label']}={a['value']}" for a in lst["responses"][-1]["answers"]))
st, _ = call("GET", f"/forms/{fid}/responses")
check("listagem sem token -> 401", st, 401)

print("== draft não aceita resposta ==")
call("POST", f"/forms/{fid}/unpublish", token=tok)
st, _ = call("GET", f"/public/forms/{slug}")
check("slug de draft -> 404", st, 404)
st, _ = call("POST", f"/public/forms/{slug}/responses", {"answers": {}})
check("submit em draft -> 404", st, 404)
st, rep = call("POST", f"/forms/{fid}/publish", token=tok)
check("republicar", st, 200)

print("== respostas sobrevivem à edição da definição ==")
call("PUT", f"/forms/{fid}", {"title": "Pesquisa Smoke", "description": "editada",
     "fields": [{"label": "Nome completo", "type": "text", "required": True}]}, token=tok)
st, lst2 = call("GET", f"/forms/{fid}/responses", token=tok)
check("respostas preservadas após editar campos", lst2["total"], 3)

print("== isolamento entre usuários ==")
call("POST", "/auth/register", {"email": "outro@example.com", "password": "senha12345"})
st, s2 = call("POST", "/auth/login", {"email": "outro@example.com", "password": "senha12345"})
tok2 = s2["token"]
st, _ = call("GET", f"/forms/{fid}", token=tok2)
check("formulário de outro usuário -> 404", st, 404)
st, _ = call("GET", f"/forms/{fid}/responses", token=tok2)
check("respostas de outro usuário -> 404", st, 404)

call("DELETE", f"/forms/{fid}", token=tok)
print(f"\n{ok} ok, {fail} falha(s)")
sys.exit(1 if fail else 0)
