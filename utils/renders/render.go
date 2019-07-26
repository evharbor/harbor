package renders

import (
	"github.com/flosch/pongo2"
	"github.com/gin-gonic/gin"
)

// HTMLRenderByPongo2 response html render by template engin pongo2
func HTMLRenderByPongo2(ctx *gin.Context, code int, filename string, data pongo2.Context) {

	contentType := "text/html; charset=utf-8"
	var template *pongo2.Template
	if gin.Mode() == "debug" {
		template = pongo2.Must(pongo2.FromFile(filename))
	} else {
		template = pongo2.Must(pongo2.FromCache(filename))
	}

	html, err := template.Execute(data)
	if err != nil {
		ctx.String(500, "render template error:"+err.Error())
		return
	}
	ctx.Data(code, contentType, []byte(html))
}
