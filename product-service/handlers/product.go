package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"log/slog"
	"net/http"
	"product-service/internal/products"
	"product-service/pkg/ctxmanage"
	"product-service/pkg/logkey"
)

// TODO: Return all the products so user can add specific to cart
type handler struct {
	// Conn is a dependency for handlers package,
	//adding it in the struct so handler package method can call method using conn struct
	//models.Conn
	products.Conf // using a struct that wraps interface instead of using conn type directly

}

func (h *handler) CreateProduct(c *gin.Context) {

	// Get the traceId from the request for tracking logs
	traceId := ctxmanage.GetTraceIdOfRequest(c)

	// Check if the size of the request body exceeds 5 KB
	if c.Request.ContentLength > 5*1024 {
		slog.Error("request body limit breached", slog.String("TRACE ID", traceId), slog.Int64("Size Received", c.Request.ContentLength))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Request body too large."})
		return
	}

	// Variable to store the decoded request body
	var newProduct products.NewProduct

	// Bind JSON payload to the newProduct struct
	err := c.ShouldBindJSON(&newProduct)
	if err != nil {
		slog.Error("json validation error", slog.String("TRACE ID", traceId), slog.String("Error", err.Error()))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": http.StatusText(http.StatusBadRequest)})
		return
	}

	// Use the validator package to validate the struct
	validate := validator.New()
	err = validate.Struct(newProduct)

	// Check for validation errors
	if err != nil {
		var vErrs validator.ValidationErrors
		if errors.As(err, &vErrs) {
			for _, vErr := range vErrs {
				switch vErr.Tag() {
				case "required":
					slog.Error("validation failed", slog.String("TRACE ID", traceId), slog.String("Error", err.Error()))
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": vErr.Field() + " value missing"})
					return
				case "min":
					slog.Error("validation failed", slog.String("TRACE ID", traceId), slog.String("Error", err.Error()))
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": vErr.Field() + " value is less than " + vErr.Param()})
					return
				default:
					slog.Error("validation failed", slog.String("TRACE ID", traceId), slog.String("Error", err.Error()))
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": http.StatusText(http.StatusBadRequest)})
					return
				}
			}
		}

		// Log validation errors
		slog.Error("validation failed", slog.String("TRACE ID", traceId), slog.String("ERROR", err.Error()))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": http.StatusText(http.StatusBadRequest)})
		return
	}

	// Call InsertProduct to save product to the database
	insertedProduct, err := h.InsertProduct(c.Request.Context(), newProduct)
	if err != nil {
		slog.Error("error in inserting the product", slog.String(logkey.TraceID, traceId), slog.String("Error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Product Creation Failed"})
		return
	}

	// Start a goroutine to create Stripe product and price
	// Safely log errors in case of failure during Stripe integration
	go func(product products.Product) {

		err := h.CreateProductPriceStripe(product)
		if err != nil {
			slog.Error("error in creating product price in Stripe", slog.String("Trace ID", traceId), slog.String("ProductID", product.ID), slog.String("Error", err.Error()))
		}
	}(insertedProduct)

	// Respond with the inserted product
	c.JSON(http.StatusOK, insertedProduct)
}

func (h *handler) ProductStockAndStripePriceId(c *gin.Context) {

	// Get the traceId from the request for tracking logs
	traceId := ctxmanage.GetTraceIdOfRequest(c)

	// Extract the product ID from the URL parameters
	productID := c.Param("productID")
	if productID == "" {
		slog.Error("missing product id", slog.String(logkey.TraceID, traceId))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Product ID is required"})
		return
	}

	// Call GetProductStockAndPrice to retrieve stock and price for the product
	stock, priceID, err := h.Conf.GetProductStockAndStripePriceId(c.Request.Context(), productID)
	if err != nil {
		// Log the error
		slog.Error("error in fetching product stock and price", slog.String(logkey.TraceID, traceId),
			slog.String(logkey.ERROR, err.Error()), slog.String("ProductID", productID))

		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to retrieve product stock and price"})
		return
	}

	// Log the success operation
	slog.Info("successfully retrieved product stock and price", slog.String("Trace ID", traceId), slog.String("ProductID", productID), slog.Int("Stock", stock), slog.String("PriceID", priceID))

	// Respond with the stock and price
	c.JSON(http.StatusOK, gin.H{"product_id": productID, "stock": stock, "price_id": priceID})
}
