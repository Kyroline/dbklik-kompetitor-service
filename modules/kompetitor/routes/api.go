// Package routes maps the kompetitor module's HTTP endpoints onto its
// controllers, kept separate from wiring (infrastructure/provider) so the
// route table itself is easy to review.
//
// Paths mirror the Laravel routes they replace (routes/web.php), so the
// proxy on the portal side is a one-liner per endpoint.
package routes

import (
	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/presentation/http/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterAPI(api *gin.RouterGroup, ctl *controllers.KompetitorController) {
	kompetitor := api.Group("/kompetitor")
	{
		// Reference data for the Riset Produk page (was the view payload).
		kompetitor.GET("/meta", ctl.Meta)

		// Kelola Kompetitor
		kompetitor.GET("/manage/data", ctl.ManageData)
		kompetitor.POST("/manage", ctl.Store)
		kompetitor.PUT("/manage/:id", ctl.Update)
		kompetitor.DELETE("/manage/:id", ctl.Destroy)

		// Mapping kompetitor (matrix kategori × brand)
		kompetitor.GET("/mapping/matrix", ctl.MappingMatrix)
		kompetitor.GET("/mapping/cell", ctl.MappingCell)
		kompetitor.POST("/mapping/cell", ctl.MappingCellUpdate)

		// Riset Produk
		kompetitor.GET("/stats", ctl.Stats)
		kompetitor.GET("/new/data", ctl.Products)
		kompetitor.GET("/data", ctl.LegacyProducts)
		kompetitor.GET("/batches", ctl.BatchCodes)

		// Our Product
		kompetitor.GET("/our-product/filter-options", ctl.FilterOptions)
		kompetitor.GET("/our-product/data", ctl.OurProducts)

		// Ingest — rows already parsed out of the Excel file by the portal.
		kompetitor.POST("/import-product", ctl.ImportProduct)
	}
}
