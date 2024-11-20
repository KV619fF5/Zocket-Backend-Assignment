package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/lib/pq"
)

// Load environment variables
func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

// Connect to the database
func connectDB() *sql.DB {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Unable to connect to the database: %v", err)
	}
	return db
}

// Product struct
type Product struct {
	ID                 int64    `json:"id,omitempty"`
	UserID             int64    `json:"user_id"`
	ProductName        string   `json:"product_name"`
	ProductDescription string   `json:"product_description"`
	ProductImages      []string `json:"product_images"`
	ProductPrice       float64  `json:"product_price"`
}

// POST /products: Create a new product
func createProduct(c *gin.Context) {
	var product Product
	if err := c.BindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	db := connectDB()
	defer db.Close()

	query := `INSERT INTO products (user_id, product_name, product_description, product_images, product_price) 
              VALUES ($1, $2, $3, $4, $5) RETURNING id`
	var productID int64
	err := db.QueryRow(query, product.UserID, product.ProductName, product.ProductDescription, pq.Array(product.ProductImages), product.ProductPrice).Scan(&productID)
	if err != nil {
		log.Printf("Error inserting product: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product created successfully", "product_id": productID})
}

// GET /products/:id: Retrieve a product by ID
func getProductByID(c *gin.Context) {
	id := c.Param("id")
	db := connectDB()
	defer db.Close()

	var product Product
	query := `SELECT id, user_id, product_name, product_description, product_images, product_price 
              FROM products WHERE id = $1`
	row := db.QueryRow(query, id)

	err := row.Scan(&product.ID, &product.UserID, &product.ProductName, &product.ProductDescription, pq.Array(&product.ProductImages), &product.ProductPrice)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"product": product})
}

// GET /products: List all products with optional filtering
func getProducts(c *gin.Context) {
	userID := c.Query("user_id")
	minPrice := c.Query("min_price")
	maxPrice := c.Query("max_price")
	productName := c.Query("product_name")

	db := connectDB()
	defer db.Close()

	query := `SELECT id, user_id, product_name, product_description, product_images, product_price 
              FROM products WHERE 1=1`
	args := []interface{}{}

	if userID != "" {
		query += " AND user_id = $1"
		args = append(args, userID)
	}
	if minPrice != "" {
		query += " AND product_price >= $2"
		args = append(args, minPrice)
	}
	if maxPrice != "" {
		query += " AND product_price <= $3"
		args = append(args, maxPrice)
	}
	if productName != "" {
		query += " AND product_name ILIKE $4"
		args = append(args, "%"+productName+"%")
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products"})
		return
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var product Product
		err := rows.Scan(&product.ID, &product.UserID, &product.ProductName, &product.ProductDescription, pq.Array(&product.ProductImages), &product.ProductPrice)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		products = append(products, product)
	}

	c.JSON(http.StatusOK, gin.H{"products": products})
}

// Main function
func main() {
	loadEnv() // Load environment variables

	r := gin.Default()

	// Define routes
	r.POST("/products", createProduct)     // Create product
	r.GET("/products/:id", getProductByID) // Get product by ID
	r.GET("/products", getProducts)        // Get all products with filtering

	// Start the server
	r.Run(":8080")
}
