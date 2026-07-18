# ZACA-CHANGES

Cambios propios de Zacatrus sobre el fork de [`knadh/listmonk`](https://github.com/knadh/listmonk).
Objetivo: **fork mergeable**. Toda la lógica nueva vive en ficheros nuevos; donde
hay que tocar upstream, los hunks son mínimos, localizados y aditivos. **Sin
cambios de esquema ni migraciones de BD.**

- Rama `master` = espejo limpio del upstream (solo releases).
- Rama `zaca` = trabajo/deploy = upstream (arrancado desde el tag `v6.2.0`) + estos commits.
- Actualizar a una release oficial: `git fetch upstream --tags && git checkout zaca && git merge vX.Y.Z`.

---

## 1. Multiidioma (ES/FR) para piezas de cara al suscriptor

listmonk es mono-idioma por instancia: un único `*i18n.I18n` global (construido al
arranque con `app.lang`) sirve admin + páginas públicas + emails de sistema.
Aquí hacemos que **las piezas de cara al suscriptor** salgan en el idioma del
suscriptor. El **admin sigue mono-idioma** (idioma del equipo).

### Cómo se resuelve el idioma (cadena)

`attribs.lang` (override explícito, si está) → **tag `lang:xx` de la lista** del
contexto → `app.lang`.

- **Fuente principal: la lista.** Basta **etiquetar cada lista con `lang:es` /
  `lang:fr`** (panel → lista → Tags). El modelo de Zacatrus ya está segmentado
  ES/FR y todo lo de cara al suscriptor tiene contexto de lista, así que **no hace
  falta escribir nada por suscriptor**.
- **Override opcional:** si alguien (p.ej. el conector Odoo) setea
  `subscriber.attribs.lang` (campo JSON libre, sin migración), ese valor gana.

Por página: gestión/baja → tag de las listas del suscriptor; opt-in (página y
email) → tag de las listas del `?l=`/del alta; formulario → `app.lang` (sin
contexto de suscriptor). `normLang` normaliza (`es-ES`→`es`).

**Alcance:** página pública de gestión/baja (`subscription.html`), opt-in
(`optin.html`), formulario público (`subscription-form.html`) y el **email de
doble opt-in**. Fuera de alcance: archive/home (sin contexto de suscriptor →
siguen en `app.lang`).

### Fichero NUEVO
- **`cmd/i18n_zaca.go`** — Toda la lógica nueva:
  - `i18nStore`: caché lazy de `*i18n.I18n` por idioma (reutiliza el
    `getI18nLang` de upstream: base inglés + overlay del idioma) y de sets de
    plantillas de email por idioma. Idioma vacío/desconocido → default (`app.lang`).
  - Resolución de idioma: `subLang(sub)` lee el override `attribs.lang`;
    `listTagLang`/`listsLang`/`subsLang` derivan del tag `lang:xx` de la(s)
    lista(s); `subUUIDLang(subUUID)` aplica la cadena completa cargando
    suscriptor+listas; `normLang` normaliza (`es-ES`→`es`).
  - Constante `zacaI18nKey` (clave de `echo.Context`).

### Hunks mínimos en upstream
- **`cmd/main.go`** — (a) campo `zi *i18nStore` en el struct `App`; (b) construir
  `zi = newI18nStore(fs, urlCfg, i18n)` antes del hook de opt-in; (c) pasar `zi` a
  `makeOptinNotifyHook(...)`; (d) asignar `zi` en el `App{...}`.
- **`cmd/public.go`** — (a) `tplRenderer.Render`: si el handler dejó una instancia
  por-petición en `c.Get(zacaI18nKey)`, usarla para el campo `.L`; si no, el global
  (comportamiento original intacto). (b) En `SubscriptionPage` (resuelve con las
  suscripciones ya cargadas y localiza también el título/mensajes),
  `SubscriptionPrefs` (`a.subUUIDLang`) y `OptinPage` (`listsLang`): `c.Set(zacaI18nKey,
  a.zi.For(...))` tras resolver el idioma.
- **`cmd/subscribers.go`** — `makeOptinNotifyHook`: nuevo parámetro `zi *i18nStore`;
  resuelve el idioma del suscriptor y renderiza asunto + plantilla del email en ese
  idioma vía `notifs.NotifyWithTpls`.
- **`internal/notifs/notifs.go`** — `NotifyWithTpls(tpls, ...)` (aditivo): permite
  renderizar con un set de plantillas por idioma. `Notify(...)` ahora delega en él
  con el set global (sin cambio de comportamiento).

### Plantillas públicas (edición mecánica)
Las plantillas de suscriptor traducían con el funcmap **global** `{{ L.T }}` (ligado
en parse-time → no varía por petición). Se cambian a `{{ $.L.T }}` (campo `.L` del
`tplData`, resuelto por petición). Se usa `$.L` (raíz) y no `.L` para que funcione
también dentro de bloques `{{ range }}` (donde el punto se rebindea).
- `static/public/templates/subscription.html` (17), `optin.html` (4),
  `subscription-form.html` (8).
- `static/public/templates/index.html` — header/footer compartidos: 2 strings
  (`archiveTitle` del RSS y `poweredBy` del pie) a `$.L.T`, para que el pie de la
  página del suscriptor quede en su idioma. archive/home no setean idioma → `app.lang`.

Las plantillas de email (`static/email-templates/*.html`) **no se tocan**: el set se
parsea por idioma con el funcmap `L` ya ligado al idioma correcto, así que `{{ L.Ts }}`
resuelve solo (header/footer del email incluidos).

---

## 2. Build / deploy: imagen propia

### Fichero NUEVO
- **`Dockerfile.zaca`** — Imagen multi-stage que **compila desde fuente** (el
  `Dockerfile` oficial solo copia un binario ya empaquetado). Stage build: `node:20`
  + Go 1.26.1 → `make dist`. Stage run: idéntico al oficial (alpine + entrypoint).
  La versión se pasa por `--build-arg LISTMONK_VERSION` (el `.git` está excluido por
  `.dockerignore`).
- **`Dockerfile.zaca.dockerignore`** — ignore de contexto específico (BuildKit lo
  prioriza sobre `.dockerignore`). Igual que el de upstream pero **manteniendo
  `**/.gitignore`** en el contexto: el `prebuild` del frontend hace
  `eslint --ignore-path .gitignore src`, que aborta con ENOENT si `frontend/.gitignore`
  se excluye. No se toca el `.dockerignore` de upstream (mergeable).

Flujo de deploy (sin registry ni CI):
```bash
docker build -f Dockerfile.zaca --build-arg LISTMONK_VERSION=v6.2.0-zaca \
  -t listmonk-zaca:v6.2.0-zaca .
docker save listmonk-zaca:v6.2.0-zaca | ssh ubuntu@news 'docker load'
# en news: /opt/listmonk/docker-compose.yml -> image: listmonk-zaca:v6.2.0-zaca
ssh ubuntu@news 'cd /opt/listmonk && docker compose up -d'
```
Rollback = apuntar `image:` al tag anterior. Se reutiliza el stack existente
(Postgres oficial + `.env`); sin migraciones propias.

---

## Verificación

1. Local (`make run`): suscriptores con `attribs {"lang":"fr"}` / `{"lang":"es"}` →
   la página pública de gestión/baja sale en FR / ES; sin `lang` → `app.lang`. Admin
   sin cambios.
2. Email de doble opt-in en el idioma del suscriptor.
3. Mergeabilidad: `git merge upstream/nightly` (o el siguiente tag) sin conflictos
   relevantes; `git diff v6.2.0..zaca --stat` pequeño y acorde a este documento.
4. `docker build -f Dockerfile.zaca ...` construye sin error.
