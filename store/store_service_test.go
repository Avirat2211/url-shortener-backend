package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var testStoreService = &StorageService{}

func init() {
	testStoreService = InitializeStore()
}

func testStoreInit(t *testing.T) {
	assert.True(t, testStoreService.redisClient != nil)
}

func TestCRUD(t *testing.T) {
	initialLink := "https://google.com/"
	userUUId := "avirat2211"
	shortURL := "BDZ9xxxs"
	SaveUrlMapping(shortURL, initialLink, userUUId)
	retrievedUrl, _, _ := RetriveInitialUrl(shortURL)
	assert.Equal(t, initialLink, retrievedUrl)
}
