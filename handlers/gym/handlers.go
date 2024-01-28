package gym

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/Grapple-2024/backend/mongo"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	MongoClient *mongo.Client
}

type Gym struct {
	ID   string `bson:"_id" json:"id"`
	Name string `bson:"name" json:"name"`
}

func (h *Handler) GetGyms(c *gin.Context) {
	ctx := c.Request.Context()

	// Find gym by ID
	gyms, err := h.MongoClient.Find(ctx, "gyms")
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
	}

	log.Info().Msgf("Find all gyms: %v", gyms)

	c.JSON(http.StatusOK, gyms)
}

func (h *Handler) GetGym(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	// Find gym by ID
	var gym Gym
	if err := h.MongoClient.FindOne(ctx, "gyms", id, &gym); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
	}

	log.Info().Msgf("Find result: %v", gym)

	c.JSON(http.StatusOK, gym)
}

func (h *Handler) CreateGym(c *gin.Context) {
	students := h.MongoClient.Collection("gyms")

	req, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.AbortWithError(http.StatusNotFound, err)
		log.Info().Msgf("Error reading request body: %v", err)
		return
	}

	body := map[string]any{}
	if err := json.Unmarshal(req, &body); err != nil {
		c.AbortWithError(http.StatusNotFound, err)
		log.Info().Msgf("Error unmarshalling json bytes to map: %v", err)
		return
	}

	res, err := students.InsertOne(c.Request.Context(), body)
	if err != nil {
		c.AbortWithError(http.StatusNotFound, err)
		log.Info().Msgf("Error inserting document: %v", err)
		return
	}

	log.Info().Msgf("Create result: %v", res)
	c.JSON(http.StatusOK, map[string]any{
		"id": res.InsertedID,
	})

}
