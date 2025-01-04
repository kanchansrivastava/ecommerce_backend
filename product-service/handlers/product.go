package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"product-service/internal/products"
	"product-service/pkg/ctxmanage"
	"product-service/pkg/logkey"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
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

func (h *handler) GetProduct(c *gin.Context) {
	// Get the traceId from the request for tracking logs
	traceId := ctxmanage.GetTraceIdOfRequest(c)

	// Extract the product ID from the URL parameter
	productID := c.Param("id")

	// Fetch the product from the database
	product, err := h.GetProductByID(c.Request.Context(), productID)
	if err != nil {
		// Handle error if product not found
		if errors.Is(err, sql.ErrNoRows) {
			slog.Error("product not found", slog.String("TRACE ID", traceId), slog.String("ProductID", productID))
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		} else {
			slog.Error("error in retrieving product", slog.String("TRACE ID", traceId), slog.String("ProductID", productID), slog.String("Error", err.Error()))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product"})
		}
		return
	}

	// Respond with the product details
	c.JSON(http.StatusOK, product)
}

func (h *handler) UpdateProduct(c *gin.Context) {
	// Get the traceId from the request for tracking logs
	traceId := ctxmanage.GetTraceIdOfRequest(c)

	// Extract the product ID from the URL parameter
	productID := c.Param("id")
	if productID == "" {
		slog.Error("missing product ID in request", slog.String("TRACE ID", traceId))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Product ID is required"})
		return
	}

	// Fetch the current product from the database
	currentProduct, err := h.GetProductByID(c.Request.Context(), productID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Handle error if product not found
			slog.Error("product not found", slog.String("TRACE ID", traceId), slog.String("ProductID", productID))
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		} else {
			// Handle database or other errors
			slog.Error("error in retrieving product", slog.String("TRACE ID", traceId), slog.String("ProductID", productID), slog.String("Error", err.Error()))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product"})
		}
		return
	}

	// Variable to store the updated product
	var updatedProduct products.Product

	// Bind JSON payload to the updatedProduct struct
	if err = c.ShouldBindJSON(&updatedProduct); err != nil {
		slog.Error("json validation error", slog.String("TRACE ID", traceId), slog.String("Error", err.Error()))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload"})
		return
	}

	// Validate the updated product data
	validate := validator.New()
	if err = validate.Struct(updatedProduct); err != nil {
		var validationErrors []string
		if errs, ok := err.(validator.ValidationErrors); ok {
			for _, fieldErr := range errs {
				validationErrors = append(validationErrors, fmt.Sprintf("%s: %s", fieldErr.Field(), fieldErr.Tag()))
			}
		} else {
			validationErrors = append(validationErrors, "Validation failed")
		}

		slog.Error("validation failed", slog.String("TRACE ID", traceId), slog.String("Error", strings.Join(validationErrors, ", ")))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": validationErrors})
		return
	}

	// Preserve immutable fields
	updatedProduct.ID = productID
	updatedProduct.CreatedAt = currentProduct.CreatedAt // Ensure creation time isn't altered

	// Call UpdateProductInDB to update product details in the database
	product, err := h.UpdateProductInDB(c.Request.Context(), productID, updatedProduct)
	if err != nil {
		slog.Error("error in updating the product", slog.String("TRACE ID", traceId), slog.String("ProductID", productID), slog.String("Error", err.Error()))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Product update failed"})
		return
	}

	// Log successful update
	slog.Info("product updated successfully", slog.String("TRACE ID", traceId), slog.String("ProductID", productID))

	// Respond with the updated product
	c.JSON(http.StatusOK, gin.H{"message": "Product updated successfully", "product": product})
}

func (h *handler) DeleteProduct(c *gin.Context) {
	// Get the traceId from the request for tracking logs
	traceId := ctxmanage.GetTraceIdOfRequest(c)

	// Extract the product ID from the URL parameter
	productID := c.Param("id")

	// Check if the product exists
	_, err := h.GetProductByID(c.Request.Context(), productID)
	if err != nil {
		// Handle error if product not found
		if errors.Is(err, sql.ErrNoRows) {
			slog.Error("product not found", slog.String("TRACE ID", traceId), slog.String("ProductID", productID))
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		} else {
			slog.Error("error in retrieving product", slog.String("TRACE ID", traceId), slog.String("ProductID", productID), slog.String("Error", err.Error()))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product"})
		}
		return
	}

	// Call DeleteProduct to remove the product from the database
	err = h.DeleteProductFromDB(c.Request.Context(), productID)
	if err != nil {
		slog.Error("error in deleting the product", slog.String(logkey.TraceID, traceId), slog.String("ProductID", productID), slog.String("Error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Product deletion failed"})
		return
	}

	// Respond with a success message
	c.JSON(http.StatusOK, gin.H{"message": "Product successfully deleted"})
}

func (h *handler) ListProducts(c *gin.Context) {
	// Get the traceId from the request for tracking logs
	traceId := ctxmanage.GetTraceIdOfRequest(c)

	// Optional query parameters for filtering, pagination, and sorting
	nameFilter := c.Query("name")           // Filter by name
	categoryFilter := c.Query("category")   // Filter by category
	limit := c.DefaultQuery("limit", "10")  // Default limit is 10
	offset := c.DefaultQuery("offset", "0") // Default offset is 0
	sort := c.DefaultQuery("sort", "name")  // Default sort is by name
	order := c.DefaultQuery("order", "asc") // Default order is ascending

	// Parse limit and offset into integers
	limitInt, err := strconv.Atoi(limit)
	if err != nil || limitInt <= 0 {
		slog.Error("invalid limit parameter", slog.String("TRACE ID", traceId))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter"})
		return
	}
	offsetInt, err := strconv.Atoi(offset)
	if err != nil || offsetInt < 0 {
		slog.Error("invalid offset parameter", slog.String("TRACE ID", traceId))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid offset parameter"})
		return
	}

	// Call ListProductsFromDB to fetch products from the database
	products, err := h.ListProductsFromDB(c.Request.Context(), nameFilter, categoryFilter, limitInt, offsetInt, sort, order)
	if err != nil {
		slog.Error("error in fetching products", slog.String("TRACE ID", traceId), slog.String("Error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products"})
		return
	}

	// Respond with the list of products
	c.JSON(http.StatusOK, gin.H{"products": products})
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
