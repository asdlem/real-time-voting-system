package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"realtime-voting-backend/models"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestCreatePoll(t *testing.T) {
	router, db := SetupTestEnvironment(t)
	// Clear tables before test
	ClearTables(db)

	// Prepare request body
	pollData := gin.H{
		"question":  "Unit Test Poll?",
		"poll_type": 0,
		"options": []gin.H{
			{"text": "Yes"},
			{"text": "No"},
		},
	}
	jsonData, _ := json.Marshal(pollData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/polls", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	// Check response body
	var createdPoll models.Poll
	err := json.Unmarshal(w.Body.Bytes(), &createdPoll)
	assert.NoError(t, err)
	assert.Equal(t, "Unit Test Poll?", createdPoll.Question)
	assert.Equal(t, models.SingleChoice, createdPoll.PollType)
	assert.Len(t, createdPoll.Options, 2)
	assert.Equal(t, "Yes", createdPoll.Options[0].Text)
	assert.Equal(t, "No", createdPoll.Options[1].Text)
	assert.True(t, createdPoll.IsActive) // Check default value
	assert.NotZero(t, createdPoll.ID)
	assert.NotZero(t, createdPoll.Options[0].ID)
	assert.NotZero(t, createdPoll.Options[1].ID)
	assert.Equal(t, createdPoll.ID, createdPoll.Options[0].PollID)
}

func TestCreatePoll_InvalidInput(t *testing.T) {
	router, _ := SetupTestEnvironment(t)

	tests := []struct {
		name         string
		body         gin.H
		expectedCode int
		expectedErr  string
	}{
		{
			name: "Missing question",
			body: gin.H{
				"poll_type": 0,
				"options":   []gin.H{{"text": "A"}, {"text": "B"}},
			},
			expectedCode: http.StatusBadRequest,
			expectedErr:  "Key: 'CreatePollInput.Question' Error:Field validation for 'Question' failed on the 'required' tag", // Gin validation error
		},
		{
			name: "Missing options",
			body: gin.H{
				"question":  "Q?",
				"poll_type": 0,
			},
			expectedCode: http.StatusBadRequest,
			expectedErr:  "Key: 'CreatePollInput.Options' Error:Field validation for 'Options' failed on the 'required' tag",
		},
		{
			name: "Not enough options",
			body: gin.H{
				"question":  "Q?",
				"poll_type": 0,
				"options":   []gin.H{{"text": "A"}},
			},
			expectedCode: http.StatusBadRequest,
			expectedErr:  "Key: 'CreatePollInput.Options' Error:Field validation for 'Options' failed on the 'min' tag", // Gin handles min length validation
		},
		{
			name: "Option text missing",
			body: gin.H{
				"question":  "Q?",
				"poll_type": 0,
				"options":   []gin.H{{"text": "A"}, {"text": ""}},
			},
			expectedCode: http.StatusBadRequest,
			expectedErr:  "Key: 'CreatePollInput.Options[1].Text' Error:Field validation for 'Text' failed on the 'required' tag", // Dive validation
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			jsonData, _ := json.Marshal(tc.body)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/polls", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedCode, w.Code)

			var responseBody map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &responseBody)
			assert.NoError(t, err)
			assert.Contains(t, responseBody["error"], tc.expectedErr)
		})
	}
}

func TestGetPolls(t *testing.T) {
	router, db := SetupTestEnvironment(t)
	ClearTables(db)

	// Create some polls first
	poll1 := models.Poll{Question: "Poll 1", Options: []models.PollOption{{Text: "1A"}, {Text: "1B"}}}
	poll2 := models.Poll{Question: "Poll 2", Options: []models.PollOption{{Text: "2A"}, {Text: "2B"}}}
	db.Create(&poll1)
	db.Create(&poll2)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/polls", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var polls []models.Poll
	err := json.Unmarshal(w.Body.Bytes(), &polls)
	assert.NoError(t, err)
	assert.Len(t, polls, 2)

	// Basic check (order might depend on DB/query, here assuming reversed insertion order)
	assert.Equal(t, "Poll 2", polls[0].Question)
	assert.Equal(t, "Poll 1", polls[1].Question)
	assert.Len(t, polls[0].Options, 2)
	assert.Len(t, polls[1].Options, 2)
}

func TestGetPolls_Empty(t *testing.T) {
	router, db := SetupTestEnvironment(t)
	ClearTables(db)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/polls", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var polls []models.Poll
	err := json.Unmarshal(w.Body.Bytes(), &polls)
	assert.NoError(t, err)
	assert.Len(t, polls, 0)
}

func TestGetPoll(t *testing.T) {
	router, db := SetupTestEnvironment(t)
	ClearTables(db)

	// Create a poll first
	poll := models.Poll{Question: "Specific Poll", Options: []models.PollOption{{Text: "Opt A"}, {Text: "Opt B"}}}
	db.Create(&poll)
	assert.NotZero(t, poll.ID) // Ensure ID is set
	pollID := poll.ID

	w := httptest.NewRecorder()
	url := fmt.Sprintf("/api/polls/%d", pollID)
	req, _ := http.NewRequest("GET", url, nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var fetchedPoll models.Poll
	err := json.Unmarshal(w.Body.Bytes(), &fetchedPoll)
	assert.NoError(t, err)
	assert.Equal(t, pollID, fetchedPoll.ID)
	assert.Equal(t, "Specific Poll", fetchedPoll.Question)
	assert.Len(t, fetchedPoll.Options, 2)
	assert.Equal(t, "Opt A", fetchedPoll.Options[0].Text)
	assert.Equal(t, "Opt B", fetchedPoll.Options[1].Text)
}

func TestGetPoll_NotFound(t *testing.T) {
	router, db := SetupTestEnvironment(t)
	ClearTables(db)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/polls/9999", nil) // Non-existent ID

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var responseBody map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.NoError(t, err)
	assert.Equal(t, "Poll not found", responseBody["error"])
}

func TestUpdatePoll(t *testing.T) {
	router, db := SetupTestEnvironment(t)
	ClearTables(db)

	// Create a poll first
	poll := models.Poll{Question: "Original Question", IsActive: true, Options: []models.PollOption{{Text: "A"}, {Text: "B"}}}
	db.Create(&poll)
	pollID := poll.ID

	// Prepare update data
	updatedQuestion := "Updated Question"
	updatedActive := false
	updateData := gin.H{
		"question":  &updatedQuestion,
		"is_active": &updatedActive,
	}
	jsonData, _ := json.Marshal(updateData)

	w := httptest.NewRecorder()
	url := fmt.Sprintf("/api/polls/%d", pollID)
	req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check response body
	var updatedPollResp models.Poll
	err := json.Unmarshal(w.Body.Bytes(), &updatedPollResp)
	assert.NoError(t, err)
	assert.Equal(t, pollID, updatedPollResp.ID)
	assert.Equal(t, updatedQuestion, updatedPollResp.Question)
	assert.Equal(t, updatedActive, updatedPollResp.IsActive)
	assert.Len(t, updatedPollResp.Options, 2) // Ensure options are still there

	// Verify in DB
	var pollInDB models.Poll
	db.First(&pollInDB, pollID)
	assert.Equal(t, updatedQuestion, pollInDB.Question)
	assert.Equal(t, updatedActive, pollInDB.IsActive)
}

func TestUpdatePoll_NotFound(t *testing.T) {
	router, db := SetupTestEnvironment(t)
	ClearTables(db)

	updatedQuestion := "Does not matter"
	updateData := gin.H{"question": &updatedQuestion}
	jsonData, _ := json.Marshal(updateData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/polls/9999", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeletePoll(t *testing.T) {
	router, db := SetupTestEnvironment(t)
	ClearTables(db)

	// Create a poll with options
	poll := models.Poll{Question: "To Be Deleted", Options: []models.PollOption{{Text: "Del A"}, {Text: "Del B"}}}
	db.Create(&poll)
	pollID := poll.ID
	assert.NotZero(t, poll.Options[0].ID)
	optionID1 := poll.Options[0].ID

	// Ensure poll and options exist before delete
	var countBefore int64
	db.Model(&models.Poll{}).Where("id = ?", pollID).Count(&countBefore)
	assert.Equal(t, int64(1), countBefore)
	db.Model(&models.PollOption{}).Where("poll_id = ?", pollID).Count(&countBefore)
	assert.Equal(t, int64(2), countBefore)

	w := httptest.NewRecorder()
	url := fmt.Sprintf("/api/polls/%d", pollID)
	req, _ := http.NewRequest("DELETE", url, nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check response body
	var responseBody map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.NoError(t, err)
	assert.Equal(t, "Poll deleted successfully", responseBody["message"])

	// Verify poll deletion in DB
	var countAfter int64
	db.Model(&models.Poll{}).Where("id = ?", pollID).Count(&countAfter)
	assert.Equal(t, int64(0), countAfter)

	// Verify options deletion in DB
	db.Model(&models.PollOption{}).Where("poll_id = ?", pollID).Count(&countAfter)
	assert.Equal(t, int64(0), countAfter)

	// Try getting the specific option to double-check
	var deletedOption models.PollOption
	result := db.First(&deletedOption, optionID1)
	assert.Error(t, result.Error)
	assert.ErrorIs(t, result.Error, gorm.ErrRecordNotFound)
}

func TestDeletePoll_NotFound(t *testing.T) {
	router, db := SetupTestEnvironment(t)
	ClearTables(db)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/polls/9999", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- Tests for SubmitVote --- //

func TestSubmitVote_SingleChoice(t *testing.T) {
	router, db := SetupTestEnvironment(t)
	ClearTables(db)

	// Create a single-choice poll
	poll := models.Poll{
		Question: "Single Choice Test",
		PollType: models.SingleChoice,
		IsActive: true,
		Options:  []models.PollOption{{Text: "S1"}, {Text: "S2"}},
	}
	db.Create(&poll)
	pollID := poll.ID
	optionID1 := poll.Options[0].ID
	optionID2 := poll.Options[1].ID

	// Vote for option 1
	voteData := gin.H{"option_ids": []uint{optionID1}}
	jsonData, _ := json.Marshal(voteData)
	w := httptest.NewRecorder()
	url := fmt.Sprintf("/api/polls/%d/vote", pollID)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Check vote count in DB
	var opt1 models.PollOption
	db.First(&opt1, optionID1)
	assert.Equal(t, int64(1), opt1.Votes)
	// Check response includes results
	var respBody map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &respBody)
	assert.NoError(t, err)
	assert.Equal(t, "Vote submitted successfully", respBody["message"])
	assert.NotNil(t, respBody["current_results"])

	// Try voting for option 2
	voteData = gin.H{"option_ids": []uint{optionID2}}
	jsonData, _ = json.Marshal(voteData)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	// Check counts
	db.First(&opt1, optionID1)
	var opt2 models.PollOption
	db.First(&opt2, optionID2)
	assert.Equal(t, int64(1), opt1.Votes)
	assert.Equal(t, int64(1), opt2.Votes)
}

func TestSubmitVote_MultiChoice(t *testing.T) {
	router, db := SetupTestEnvironment(t)
	ClearTables(db)

	// Create a multi-choice poll
	poll := models.Poll{
		Question: "Multi Choice Test",
		PollType: models.MultiChoice,
		IsActive: true,
		Options:  []models.PollOption{{Text: "M1"}, {Text: "M2"}, {Text: "M3"}},
	}
	db.Create(&poll)
	pollID := poll.ID
	optionID1 := poll.Options[0].ID
	optionID2 := poll.Options[1].ID
	optionID3 := poll.Options[2].ID

	// Vote for options 1 and 3
	voteData := gin.H{"option_ids": []uint{optionID1, optionID3}}
	jsonData, _ := json.Marshal(voteData)
	w := httptest.NewRecorder()
	url := fmt.Sprintf("/api/polls/%d/vote", pollID)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Check counts
	var opt1, opt2, opt3 models.PollOption
	db.First(&opt1, optionID1)
	db.First(&opt2, optionID2)
	db.First(&opt3, optionID3)
	assert.Equal(t, int64(1), opt1.Votes)
	assert.Equal(t, int64(0), opt2.Votes)
	assert.Equal(t, int64(1), opt3.Votes)
}

func TestSubmitVote_SingleChoice_MultipleOptionsSent(t *testing.T) {
	router, db := SetupTestEnvironment(t)
	ClearTables(db)

	// Create a single-choice poll
	poll := models.Poll{Question: "Single Invalid", PollType: models.SingleChoice, IsActive: true, Options: []models.PollOption{{Text: "S1"}, {Text: "S2"}}}
	db.Create(&poll)
	pollID := poll.ID
	optionID1 := poll.Options[0].ID
	optionID2 := poll.Options[1].ID

	// Try voting for both options
	voteData := gin.H{"option_ids": []uint{optionID1, optionID2}}
	jsonData, _ := json.Marshal(voteData)
	w := httptest.NewRecorder()
	url := fmt.Sprintf("/api/polls/%d/vote", pollID)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var responseBody map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.NoError(t, err)
	assert.Equal(t, "Single choice poll allows only one option", responseBody["error"])
}

func TestSubmitVote_InvalidOptionID(t *testing.T) {
	router, db := SetupTestEnvironment(t)
	ClearTables(db)

	// Create a poll
	poll := models.Poll{Question: "Invalid Option Test", IsActive: true, Options: []models.PollOption{{Text: "Valid"}}}
	db.Create(&poll)
	pollID := poll.ID
	invalidOptionID := uint(9999)

	// Vote for invalid option
	voteData := gin.H{"option_ids": []uint{invalidOptionID}}
	jsonData, _ := json.Marshal(voteData)
	w := httptest.NewRecorder()
	url := fmt.Sprintf("/api/polls/%d/vote", pollID)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var responseBody map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.NoError(t, err)
	assert.Contains(t, responseBody["error"], "Invalid option ID")
}

func TestSubmitVote_PollNotFound(t *testing.T) {
	router, db := SetupTestEnvironment(t)
	ClearTables(db)

	voteData := gin.H{"option_ids": []uint{1}} // Option ID doesn't matter here
	jsonData, _ := json.Marshal(voteData)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/polls/9999/vote", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSubmitVote_PollInactive(t *testing.T) {
	router, db := SetupTestEnvironment(t)
	ClearTables(db)

	// Create an inactive poll
	poll := models.Poll{Question: "Inactive Poll", IsActive: false, Options: []models.PollOption{{Text: "Opt"}}}
	db.Create(&poll)
	pollID := poll.ID
	optionID := poll.Options[0].ID

	voteData := gin.H{"option_ids": []uint{optionID}}
	jsonData, _ := json.Marshal(voteData)
	w := httptest.NewRecorder()
	url := fmt.Sprintf("/api/polls/%d/vote", pollID)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	var responseBody map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &responseBody)
	assert.NoError(t, err)
	assert.Equal(t, "Voting on this poll is closed", responseBody["error"])
}
