package main

import (
	"strings"

	"github.com/labstack/echo/v4"
)

// zacaExcludeTagsParam devuelve las etiquetas que el panel pide ocultar con
// ?exclude_tag=x (repetible, igual que ?tag=). Sin el parámetro no se excluye
// nada, así que el API se comporta exactamente como el de upstream.
func zacaExcludeTagsParam(c echo.Context) []string {
	return c.QueryParams()["exclude_tag"]
}

// zacaTestPrefix es lo que se antepone al asunto de un envío de prueba.
const zacaTestPrefix = "[TEST] "

// zacaTestSubject antepone «[TEST] » al asunto de un envío de prueba
// (POST /api/campaigns/:id/test), para que quien lo recibe sepa que no es el
// envío de verdad. El envío real de la campaña y las vistas previas no pasan
// por aquí. Si el asunto ya empieza por [TEST] no se repite. El asunto se
// compila después como plantilla, así que un {{ ... }} sigue funcionando.
func zacaTestSubject(subject string) string {
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(subject)), "[TEST]") {
		return subject
	}
	return zacaTestPrefix + subject
}
