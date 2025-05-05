package handler

import (
	"net/http"
	"strconv"

	"api-gateway/service"

	inventorypb "github.com/Kroph/Programming/proto/inventory"
	orderpb "github.com/Kroph/Programming/proto/order"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	grpcClients *service.GrpcClients
	authService service.AuthService
}

func NewHandler(grpcClients *service.GrpcClients, authService service.AuthService) *Handler {
	return &Handler{
		grpcClients: grpcClients,
		authService: authService,
	}
}

// User handlers

func (h *Handler) RegisterUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.grpcClients.RegisterUser(c.Request.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	token, err := h.authService.GenerateToken(user.Id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"user": gin.H{
			"id":       user.Id,
			"username": user.Username,
			"email":    user.Email,
		},
		"token": token,
	})
}

func (h *Handler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authResponse, err := h.grpcClients.AuthenticateUser(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := h.authService.GenerateToken(authResponse.UserId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"user": gin.H{
			"id":       authResponse.UserId,
			"username": authResponse.Username,
			"email":    authResponse.Email,
		},
		"token": token,
	})
}

func (h *Handler) GetUserProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	profile, err := h.grpcClients.GetUserProfile(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":         profile.Id,
			"username":   profile.Username,
			"email":      profile.Email,
			"created_at": profile.CreatedAt,
		},
	})
}

// Product handlers

func (h *Handler) CreateProduct(c *gin.Context) {
	var req struct {
		Name        string  `json:"name" binding:"required"`
		Description string  `json:"description"`
		Price       float64 `json:"price" binding:"required,gt=0"`
		Stock       int32   `json:"stock" binding:"required,gte=0"`
		CategoryID  string  `json:"category_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product, err := h.grpcClients.CreateProduct(c.Request.Context(), &inventorypb.CreateProductRequest{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		CategoryId:  req.CategoryID,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, product)
}

func (h *Handler) GetProduct(c *gin.Context) {
	id := c.Param("id")

	product, err := h.grpcClients.GetProduct(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	c.JSON(http.StatusOK, product)
}

func (h *Handler) UpdateProduct(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
		Stock       int32   `json:"stock"`
		CategoryID  string  `json:"category_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product, err := h.grpcClients.UpdateProduct(c.Request.Context(), &inventorypb.UpdateProductRequest{
		Id:          id,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		CategoryId:  req.CategoryID,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, product)
}

func (h *Handler) DeleteProduct(c *gin.Context) {
	id := c.Param("id")

	response, err := h.grpcClients.DeleteProduct(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": response.Success,
		"message": response.Message,
	})
}

func (h *Handler) ListProducts(c *gin.Context) {
	var filter inventorypb.ProductFilter

	if categoryID := c.Query("category_id"); categoryID != "" {
		filter.CategoryId = categoryID
	}

	if minPrice := c.Query("min_price"); minPrice != "" {
		if minPriceFloat, err := strconv.ParseFloat(minPrice, 64); err == nil {
			filter.MinPrice = minPriceFloat
		}
	}

	if maxPrice := c.Query("max_price"); maxPrice != "" {
		if maxPriceFloat, err := strconv.ParseFloat(maxPrice, 64); err == nil {
			filter.MaxPrice = maxPriceFloat
		}
	}

	if inStock := c.Query("in_stock"); inStock == "true" {
		filter.InStock = true
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	filter.Page = int32(page)
	filter.PageSize = int32(pageSize)

	response, err := h.grpcClients.ListProducts(c.Request.Context(), &filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// Category handlers

func (h *Handler) CreateCategory(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	category, err := h.grpcClients.CreateCategory(c.Request.Context(), &inventorypb.CreateCategoryRequest{
		Name:        req.Name,
		Description: req.Description,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, category)
}

func (h *Handler) GetCategory(c *gin.Context) {
	id := c.Param("id")

	category, err := h.grpcClients.GetCategory(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	c.JSON(http.StatusOK, category)
}

func (h *Handler) UpdateCategory(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	category, err := h.grpcClients.UpdateCategory(c.Request.Context(), &inventorypb.UpdateCategoryRequest{
		Id:          id,
		Name:        req.Name,
		Description: req.Description,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, category)
}

func (h *Handler) DeleteCategory(c *gin.Context) {
	id := c.Param("id")

	response, err := h.grpcClients.DeleteCategory(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": response.Success,
		"message": response.Message,
	})
}

func (h *Handler) ListCategories(c *gin.Context) {
	response, err := h.grpcClients.ListCategories(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// Order handlers

func (h *Handler) CreateOrder(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Items []struct {
			ProductID string  `json:"product_id" binding:"required"`
			Name      string  `json:"name" binding:"required"`
			Price     float64 `json:"price" binding:"required,gt=0"`
			Quantity  int32   `json:"quantity" binding:"required,gt=0"`
		} `json:"items" binding:"required,dive"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var orderItems []*orderpb.OrderItemRequest
	for _, item := range req.Items {
		orderItems = append(orderItems, &orderpb.OrderItemRequest{
			ProductId: item.ProductID,
			Name:      item.Name,
			Price:     item.Price,
			Quantity:  item.Quantity,
		})
	}

	// Check stock availability
	var productQuantities []*inventorypb.ProductQuantity
	for _, item := range req.Items {
		productQuantities = append(productQuantities, &inventorypb.ProductQuantity{
			ProductId: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	stockCheck, err := h.grpcClients.CheckStock(c.Request.Context(), productQuantities)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check stock: " + err.Error()})
		return
	}

	if !stockCheck.Available {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "Some items are out of stock",
			"unavailable_items": stockCheck.UnavailableItems,
		})
		return
	}

	order, err := h.grpcClients.CreateOrder(c.Request.Context(), &orderpb.CreateOrderRequest{
		UserId: userID.(string),
		Items:  orderItems,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, order)
}

func (h *Handler) GetOrder(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")

	order, err := h.grpcClients.GetOrder(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Ensure the order belongs to the authenticated user
	if order.UserId != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, order)
}

func (h *Handler) UpdateOrderStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")

	// First, check if the order belongs to the user
	order, err := h.grpcClients.GetOrder(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if order.UserId != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var status orderpb.OrderStatus
	switch req.Status {
	case "pending":
		status = orderpb.OrderStatus_PENDING
	case "paid":
		status = orderpb.OrderStatus_PAID
	case "shipped":
		status = orderpb.OrderStatus_SHIPPED
	case "delivered":
		status = orderpb.OrderStatus_DELIVERED
	case "cancelled":
		status = orderpb.OrderStatus_CANCELLED
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		return
	}

	updatedOrder, err := h.grpcClients.UpdateOrderStatus(c.Request.Context(), &orderpb.UpdateOrderStatusRequest{
		Id:     id,
		Status: status,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedOrder)
}

func (h *Handler) ListUserOrders(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	response, err := h.grpcClients.GetUserOrders(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// Register all routes
func RegisterRoutes(router *gin.Engine, h *Handler) {
	// Public routes
	auth := router.Group("/api/v1/auth")
	{
		auth.POST("/register", h.RegisterUser)
		auth.POST("/login", h.Login)
	}

	// Protected routes
	api := router.Group("/api/v1")
	api.Use(service.AuthMiddleware(h.authService))
	{
		// User routes
		api.GET("/users/profile", h.GetUserProfile)

		// Product routes
		products := api.Group("/products")
		{
			products.POST("", h.CreateProduct)
			products.GET("", h.ListProducts)
			products.GET("/:id", h.GetProduct)
			products.PUT("/:id", h.UpdateProduct)
			products.DELETE("/:id", h.DeleteProduct)
		}

		// Category routes
		categories := api.Group("/categories")
		{
			categories.POST("", h.CreateCategory)
			categories.GET("", h.ListCategories)
			categories.GET("/:id", h.GetCategory)
			categories.PUT("/:id", h.UpdateCategory)
			categories.DELETE("/:id", h.DeleteCategory)
		}

		// Order routes
		orders := api.Group("/orders")
		{
			orders.POST("", h.CreateOrder)
			orders.GET("", h.ListUserOrders)
			orders.GET("/:id", h.GetOrder)
			orders.PATCH("/:id/status", h.UpdateOrderStatus)
		}
	}
}
