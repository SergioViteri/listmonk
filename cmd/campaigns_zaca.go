package main

import "github.com/labstack/echo/v4"

// zacaExcludeTagsParam devuelve las etiquetas que el panel pide ocultar con
// ?exclude_tag=x (repetible, igual que ?tag=). Sin el parámetro no se excluye
// nada, así que el API se comporta exactamente como el de upstream.
func zacaExcludeTagsParam(c echo.Context) []string {
	return c.QueryParams()["exclude_tag"]
}
