// A simple image manipulation API using go fiber and bimg
package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/gofiber/contrib/swagger"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
	"gopkg.in/h2non/bimg.v1"
)

// Default values for image manipulation
const (
	defaultWidth  = 300
	defaultHeight = 300
	defaultAngle  = 180

	// Error variables for common error scenarios
	errUnknown = "some unknown error occured"
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("error loading .env file:", err)
	}

	// fiber Config
	fiberConfig := fiber.Config{
		// Custom error handler to provide consistent error responses
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError

			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}

			return c.Status(code).JSON(&fiber.Map{"error": err.Error()})
		}}

	// initialize fiber app
	app := fiber.New(fiberConfig)

	// Swagger Config
	swaggerConfig := swagger.Config{
		Title: "Image Manipulation API",
	}

	// Added recover, logger middleware & swagger api docs at /docs
	app.Use(logger.New(), swagger.New(swaggerConfig), recover.New())

	// Test
	app.Get("/ping", func(c *fiber.Ctx) error {
		return c.SendString("pong")
	})

	// Register route handlers
	app.Post("/rotate", Rotate)
	app.Post("/resize", Resize)
	app.Post("/grayscale", Grayscale)

	// Start the server
	port := ":" + os.Getenv("PORT")
	log.Fatal(app.Listen(port))
}

// handleRotate processes the rotate operation on the uploaded image.
// It accepts an 'angle' form parameter (90, 180 or 270 degrees).
func Rotate(c *fiber.Ctx) error {
	angle, err := strconv.Atoi(c.FormValue("angle", strconv.Itoa(defaultAngle)))

	if err != nil || (angle != 90 && angle != 180 && angle != 270) {
		return fiber.NewError(fiber.StatusBadRequest, "angle value must be 90, 180 or 270")
	}

	return processImage(c, func(img *bimg.Image) ([]byte, error) {
		return img.Rotate(bimg.Angle(angle))
	})
}

// handleResize processes the resize operation on the uploaded image.
// It accepts 'width' and 'height' form parameters.
func Resize(c *fiber.Ctx) error {
	width, err := strconv.Atoi(c.FormValue("width", strconv.Itoa(defaultWidth)))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid width value")
	}

	height, err := strconv.Atoi(c.FormValue("height", strconv.Itoa(defaultHeight)))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid height value")
	}

	return processImage(c, func(img *bimg.Image) ([]byte, error) {
		return img.Resize(width, height)
	})
}

// handleGrayscale processes the grayscale operation on the uploaded image.
func Grayscale(c *fiber.Ctx) error {
	return processImage(c, func(img *bimg.Image) ([]byte, error) {
		return img.Colourspace(bimg.InterpretationBW)
	})
}

// processImage is a helper function that handles the common flow of image processing operations.
// It takes a Fiber context and an operation function as parameters.
func processImage(c *fiber.Ctx, operation func(*bimg.Image) ([]byte, error)) error {
	buffer, err := getImageBuffer(c)

	if err != nil {
		log.Println("error occured:", err)
		return fiber.NewError(fiber.StatusInternalServerError, errUnknown)
	}

	img := bimg.NewImage(buffer)
	processed, err := operation(img)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "error processing image")
	}

	c.Set(fiber.HeaderContentType, "image/"+bimg.DetermineImageTypeName(processed))
	return c.Send(processed)
}

// getImageBuffer is a helper function that extracts the image file
// from the form-data request and returns it as a byte slice.
func getImageBuffer(c *fiber.Ctx) ([]byte, error) {
	file, err := c.FormFile("image")
	if err != nil {
		return nil, fmt.Errorf("error retrieving image file: %w", err)
	}

	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("error opening image file: %w", err)
	}
	defer src.Close()

	return io.ReadAll(src)
}
