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

**Mensajes de las páginas de confirmación/error (`message.html`).** Esas páginas
(`makeMsgTpl`) reciben los textos **ya traducidos** desde el handler con `a.i18n`
(global), así que no los cubría el cambio de `.L` en plantillas. Helper
`a.langOf(c)` (en `cmd/i18n_zaca.go`) devuelve la instancia por-petición del
contexto (o `app.lang`); los handlers de suscriptor la usan para construir los
mensajes: opt-in confirmado (`confirmOptinSubscription`), baja/gestión
(`SubscriptionPrefs`), opt-in (`OptinPage` título), export/wipe. `Render` también
pasa a usar `langOf`.

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

**Overrides de textos (estilo Zacatrus, tuteo — nunca "usted"/"vous").** Los JSON de
upstream (`i18n/es.json`, `i18n/fr.json`) usan trato formal. En vez de editarlos
(conflictos en merges), se aplica un **overlay parcial** propio por idioma:
`cmd/i18n-zaca/es.json` y `cmd/i18n-zaca/fr.json` (solo las claves a corregir),
embebidos con `go:embed` y cargados con `i18n.Load` sobre la instancia de cada
idioma en `i18nStore.For(lang)`. ES en tuteo; FR en tutoiement (decidido con Sergio).
Para añadir/ajustar un texto: editar esos JSON (upstream intacto).

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

## 3. Preheader (preview text) por campaña

El preheader es el fragmento que los clientes de correo muestran junto al asunto
en la bandeja de entrada. Sin él, el cliente coge lo primero que encuentra en el
`<body>`, que en las plantillas de listmonk suele ser el enlace de "ver en el
navegador" (`static/email-templates/default.tpl`). Upstream lo rechazó
(knadh/listmonk#1240, apoyada en #924), así que vive en el fork. Hasta ahora se
resolvía a mano metiendo un `div` oculto al principio del cuerpo en cada campaña
(caso real: campaña 7, Tsukimi FR, importada de Brevo).

**Sin migración:** se guarda bajo la clave `preheader` de la columna `attribs`
(JSONB) de `campaigns`, que ya existía sin usar. El API de creación/actualización
de campañas ya lee y escribe `attribs` tal cual (`queries/campaigns.sql`,
`internal/core/campaigns.go`), así que el modelo y el API no necesitaron ningún
cambio: guardar y leer el preheader por API ya funcionaba antes de este commit,
sin más que mandar `{"attribs": {"preheader": "..."}}`.

### Fichero NUEVO
- **`internal/manager/preheader_zaca.go`** — Toda la lógica de inyección:
  - `zacaCampaignPreheader(c)` lee y recorta `attribs.preheader`.
  - `zacaInjectPreheader(body, c)` inserta un elemento oculto justo después de
    `<body ...>` con el texto (escapado con `html.EscapeString`, así que una
    comilla o un `<` no pueden romper el HTML ni inyectar nada) y un relleno de
    caracteres invisibles (`&zwnj;&nbsp;` repetido) para que el cliente de correo
    no arrastre el resto del cuerpo al hueco que deja un preheader corto.
  - Es no-op si el content type es `plain` (no hay `<body>` que inyectar), si no
    hay preheader configurado (**campaña sin preheader = comportamiento idéntico
    al de hoy, nada de divs vacíos**) o si no se encuentra un tag `<body>` en el
    HTML renderizado.
- **`internal/manager/preheader_zaca_test.go`** — Cobertura de los casos de
  arriba más el escapado (`<script>`, comillas simples/dobles, `&`).

### Hunk mínimo en upstream
- **`internal/manager/message.go`** — En `CampaignMessage.render()`, tras
  ejecutar la plantilla compilada de la campaña, `m.body` pasa por
  `zacaInjectPreheader(...)` antes de usarse. Es el único punto de la base de
  código por el que pasa **todo** render de campaña (envío real vía
  `manager/pipe.go`, envío de prueba, ambos previews del navegador y el archivo
  público — todos llaman a `Manager.NewCampaignMessage`), así que un solo hook
  cubre todos los caminos sin tocar cada handler.

### Decisiones (documentadas, no automáticas)
- **Campaña que ya trae un preheader a mano en el HTML** (el caso de la 7): el
  sistema **no intenta detectarlo**. Sniffear HTML arbitrario para adivinar "esto
  es un preheader manual" es frágil (falsos positivos con cualquier otro bloque
  oculto al principio del body, falsos negativos si no usa `display:none` inline)
  y este fork no tiene infraestructura para validarlo contra casos reales. La
  regla es simple y predecible: si `attribs.preheader` está vacío, no se toca
  nada (igual que hoy); si se rellena, se inyecta siempre. Al adoptar el campo en
  una campaña que ya tenía el div manual, hay que quitar el div a mano — si no,
  quedan los dos.
- **Tipos de contenido:** aplica a `html`, `richtext`, `markdown` y `visual`
  (todas acaban siendo un documento HTML con `<body>`). No aplica a `plain`: no
  hay preheader en texto plano.
- **Preview y envío de prueba usan el `attribs` ya guardado en BD**, igual que el
  resto de `attribs` hoy — no hay ningún camino en el código que sobrescriba
  `attribs` con lo que llega en la petición de preview/test (solo lo hacen
  `subject`, `body`, `content_type`, etc.). Para ver el preheader en un preview o
  un envío de prueba hace falta guardar la campaña primero.
- El texto del preheader **no es una plantilla**: se trata como texto literal
  (solo escapado, sin expresiones `{{ }}`), a diferencia del asunto o el cuerpo.

### Panel
- **`frontend/src/views/Campaign.vue`** — Campo de texto "Preheader" junto al
  asunto, deshabilitado para campañas en texto plano. En `onSubmit`, el valor se
  pliega dentro de `form.attribs.preheader` (o se borra la clave si se deja
  vacío), preservando cualquier otra clave que ya hubiera en el `attribs` JSON en
  bruto (pestaña "Attribs" existente). Ya no hace falta tocar esa pestaña para
  fijar el preheader, pero sigue funcionando para quien prefiera editar el JSON
  directamente.
- **`i18n/en.json`, `i18n/es.json`, `i18n/fr.json`** — Claves nuevas
  `campaigns.preheader`, `campaigns.preheaderHelp` y
  `campaigns.preheaderPlainDisabled` (etiqueta y ayuda del campo). Estas son
  claves *nuevas*, no overrides de las que ya usa upstream, así que se añaden
  directamente a los JSON de upstream sin pasar por el mecanismo de overlay de la
  sección 1 (ese mecanismo existe para evitar conflictos al pisar valores que
  upstream también toca; una clave nueva no tiene ese riesgo).

---

## Verificación

1. Local (`make run`): suscriptores con `attribs {"lang":"fr"}` / `{"lang":"es"}` →
   la página pública de gestión/baja sale en FR / ES; sin `lang` → `app.lang`. Admin
   sin cambios.
2. Email de doble opt-in en el idioma del suscriptor.
3. Mergeabilidad: `git merge upstream/nightly` (o el siguiente tag) sin conflictos
   relevantes; `git diff v6.2.0..zaca --stat` pequeño y acorde a este documento.
4. `docker build -f Dockerfile.zaca ...` construye sin error.
5. Preheader: `go test ./internal/manager/...` cubre la inyección/escapado; sin
   entorno de staging, no se ha podido verificar en un cliente de correo real cómo
   se ve el snippet de la bandeja de entrada (Gmail/Outlook/Apple Mail truncan y
   rellenan de forma distinta) — pendiente de confirmar visualmente tras el
   despliegue.
