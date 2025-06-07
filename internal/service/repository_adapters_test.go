package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/phrazzld/scry-api/internal/domain"
	"github.com/phrazzld/scry-api/internal/store"
)

// Mock implementations for testing repository adapters
type mockCardStore struct {
	createMultipleCalled    bool
	getByIDCalled           bool
	updateContentCalled     bool
	deleteCalled            bool
	getNextReviewCardCalled bool
	withTxCalled            bool
	dbCalled                bool
	withTxReturn            store.CardStore
	dbReturn                *sql.DB
}

func (m *mockCardStore) CreateMultiple(ctx context.Context, cards []*domain.Card) error {
	m.createMultipleCalled = true
	return nil
}

func (m *mockCardStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.Card, error) {
	m.getByIDCalled = true
	return &domain.Card{ID: id}, nil
}

func (m *mockCardStore) UpdateContent(ctx context.Context, id uuid.UUID, content []byte) error {
	m.updateContentCalled = true
	return nil
}

func (m *mockCardStore) Delete(ctx context.Context, id uuid.UUID) error {
	m.deleteCalled = true
	return nil
}

func (m *mockCardStore) GetNextReviewCard(ctx context.Context, userID uuid.UUID) (*domain.Card, error) {
	m.getNextReviewCardCalled = true
	return &domain.Card{ID: uuid.New(), UserID: userID}, nil
}

func (m *mockCardStore) WithTx(tx *sql.Tx) store.CardStore {
	m.withTxCalled = true
	if m.withTxReturn != nil {
		return m.withTxReturn
	}
	return &mockCardStore{}
}

func (m *mockCardStore) DB() *sql.DB {
	m.dbCalled = true
	return m.dbReturn
}

type mockStatsStore struct {
	createCalled       bool
	getCalled          bool
	getForUpdateCalled bool
	updateCalled       bool
	deleteCalled       bool
	withTxCalled       bool
	withTxReturn       store.UserCardStatsStore
}

func (m *mockStatsStore) Create(ctx context.Context, stats *domain.UserCardStats) error {
	m.createCalled = true
	return nil
}

func (m *mockStatsStore) Get(ctx context.Context, userID, cardID uuid.UUID) (*domain.UserCardStats, error) {
	m.getCalled = true
	return &domain.UserCardStats{UserID: userID, CardID: cardID}, nil
}

func (m *mockStatsStore) GetForUpdate(ctx context.Context, userID, cardID uuid.UUID) (*domain.UserCardStats, error) {
	m.getForUpdateCalled = true
	return &domain.UserCardStats{UserID: userID, CardID: cardID}, nil
}

func (m *mockStatsStore) Update(ctx context.Context, stats *domain.UserCardStats) error {
	m.updateCalled = true
	return nil
}

func (m *mockStatsStore) Delete(ctx context.Context, userID, cardID uuid.UUID) error {
	m.deleteCalled = true
	return nil
}

func (m *mockStatsStore) WithTx(tx *sql.Tx) store.UserCardStatsStore {
	m.withTxCalled = true
	if m.withTxReturn != nil {
		return m.withTxReturn
	}
	return &mockStatsStore{}
}

type mockMemoStore struct {
	createCalled            bool
	getByIDCalled           bool
	updateCalled            bool
	updateStatusCalled      bool
	findMemosByStatusCalled bool
	withTxCalled            bool
	withTxReturn            store.MemoStore
}

func (m *mockMemoStore) Create(ctx context.Context, memo *domain.Memo) error {
	m.createCalled = true
	return nil
}

func (m *mockMemoStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.Memo, error) {
	m.getByIDCalled = true
	return &domain.Memo{ID: id}, nil
}

func (m *mockMemoStore) Update(ctx context.Context, memo *domain.Memo) error {
	m.updateCalled = true
	return nil
}

func (m *mockMemoStore) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.MemoStatus) error {
	m.updateStatusCalled = true
	return nil
}

func (m *mockMemoStore) FindMemosByStatus(
	ctx context.Context,
	status domain.MemoStatus,
	limit, offset int,
) ([]*domain.Memo, error) {
	m.findMemosByStatusCalled = true
	return []*domain.Memo{{ID: uuid.New()}}, nil
}

func (m *mockMemoStore) WithTx(tx *sql.Tx) store.MemoStore {
	m.withTxCalled = true
	if m.withTxReturn != nil {
		return m.withTxReturn
	}
	return &mockMemoStore{}
}

// Card Repository Adapter Tests
func TestNewCardRepositoryAdapter(t *testing.T) {
	mockStore := &mockCardStore{}
	mockDB := &sql.DB{}

	adapter := NewCardRepositoryAdapter(mockStore, mockDB)

	assert.NotNil(t, adapter)
	assert.Implements(t, (*CardRepository)(nil), adapter)
}

func TestCardRepositoryAdapter_Delegation(t *testing.T) {
	mockStore := &mockCardStore{}
	mockDB := &sql.DB{}
	adapter := NewCardRepositoryAdapter(mockStore, mockDB)

	ctx := context.Background()
	cardID := uuid.New()
	cards := []*domain.Card{{ID: cardID}}
	content := []byte(`{"test": "data"}`)

	// Test all methods delegate to store
	t.Run("CreateMultiple delegates", func(t *testing.T) {
		err := adapter.CreateMultiple(ctx, cards)
		assert.NoError(t, err)
		assert.True(t, mockStore.createMultipleCalled)
	})

	t.Run("GetByID delegates", func(t *testing.T) {
		card, err := adapter.GetByID(ctx, cardID)
		assert.NoError(t, err)
		assert.NotNil(t, card)
		assert.True(t, mockStore.getByIDCalled)
	})

	t.Run("UpdateContent delegates", func(t *testing.T) {
		err := adapter.UpdateContent(ctx, cardID, content)
		assert.NoError(t, err)
		assert.True(t, mockStore.updateContentCalled)
	})

	t.Run("Delete delegates", func(t *testing.T) {
		err := adapter.Delete(ctx, cardID)
		assert.NoError(t, err)
		assert.True(t, mockStore.deleteCalled)
	})

	t.Run("DB returns correct database", func(t *testing.T) {
		db := adapter.DB()
		assert.Equal(t, mockDB, db)
	})
}

func TestCardRepositoryAdapter_WithTx(t *testing.T) {
	mockStore := &mockCardStore{}
	mockTxStore := &mockCardStore{}
	mockStore.withTxReturn = mockTxStore
	mockDB := &sql.DB{}
	mockTx := &sql.Tx{}

	adapter := NewCardRepositoryAdapter(mockStore, mockDB)
	txAdapter := adapter.WithTx(mockTx)

	assert.NotNil(t, txAdapter)
	assert.NotEqual(t, adapter, txAdapter) // Should be different instance
	assert.True(t, mockStore.withTxCalled)
	assert.Equal(t, mockDB, txAdapter.DB()) // DB should be preserved
}

// Stats Repository Adapter Tests
func TestNewStatsRepositoryAdapter(t *testing.T) {
	mockStore := &mockStatsStore{}

	adapter := NewStatsRepositoryAdapter(mockStore)

	assert.NotNil(t, adapter)
	assert.Implements(t, (*StatsRepository)(nil), adapter)
}

func TestStatsRepositoryAdapter_Delegation(t *testing.T) {
	mockStore := &mockStatsStore{}
	adapter := NewStatsRepositoryAdapter(mockStore)

	ctx := context.Background()
	userID := uuid.New()
	cardID := uuid.New()
	stats := &domain.UserCardStats{UserID: userID, CardID: cardID}

	t.Run("Create delegates", func(t *testing.T) {
		err := adapter.Create(ctx, stats)
		assert.NoError(t, err)
		assert.True(t, mockStore.createCalled)
	})

	t.Run("Get delegates", func(t *testing.T) {
		result, err := adapter.Get(ctx, userID, cardID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, mockStore.getCalled)
	})

	t.Run("GetForUpdate delegates", func(t *testing.T) {
		result, err := adapter.GetForUpdate(ctx, userID, cardID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, mockStore.getForUpdateCalled)
	})

	t.Run("Update delegates", func(t *testing.T) {
		err := adapter.Update(ctx, stats)
		assert.NoError(t, err)
		assert.True(t, mockStore.updateCalled)
	})
}

func TestStatsRepositoryAdapter_WithTx(t *testing.T) {
	mockStore := &mockStatsStore{}
	mockTxStore := &mockStatsStore{}
	mockStore.withTxReturn = mockTxStore
	mockTx := &sql.Tx{}

	adapter := NewStatsRepositoryAdapter(mockStore)
	txAdapter := adapter.WithTx(mockTx)

	assert.NotNil(t, txAdapter)
	assert.NotEqual(t, adapter, txAdapter)
	assert.True(t, mockStore.withTxCalled)
}

// Memo Repository Adapter Tests
func TestNewMemoRepositoryAdapter(t *testing.T) {
	mockStore := &mockMemoStore{}
	mockDB := &sql.DB{}

	adapter := NewMemoRepositoryAdapter(mockStore, mockDB)

	assert.NotNil(t, adapter)
	assert.Implements(t, (*MemoRepository)(nil), adapter)
	assert.Equal(t, mockStore, adapter.MemoStore)
	assert.Equal(t, mockDB, adapter.db)
}

func TestMemoRepositoryAdapter_Delegation(t *testing.T) {
	mockStore := &mockMemoStore{}
	mockDB := &sql.DB{}
	adapter := NewMemoRepositoryAdapter(mockStore, mockDB)

	ctx := context.Background()
	memoID := uuid.New()
	memo := &domain.Memo{ID: memoID}

	t.Run("Create delegates", func(t *testing.T) {
		err := adapter.Create(ctx, memo)
		assert.NoError(t, err)
		assert.True(t, mockStore.createCalled)
	})

	t.Run("GetByID delegates", func(t *testing.T) {
		result, err := adapter.GetByID(ctx, memoID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, mockStore.getByIDCalled)
	})

	t.Run("Update delegates", func(t *testing.T) {
		err := adapter.Update(ctx, memo)
		assert.NoError(t, err)
		assert.True(t, mockStore.updateCalled)
	})

	t.Run("DB returns correct database", func(t *testing.T) {
		db := adapter.DB()
		assert.Equal(t, mockDB, db)
	})
}

func TestMemoRepositoryAdapter_WithTx(t *testing.T) {
	mockStore := &mockMemoStore{}
	mockTxStore := &mockMemoStore{}
	mockStore.withTxReturn = mockTxStore
	mockDB := &sql.DB{}
	mockTx := &sql.Tx{}

	adapter := NewMemoRepositoryAdapter(mockStore, mockDB)
	txAdapter := adapter.WithTx(mockTx)

	assert.NotNil(t, txAdapter)
	assert.NotEqual(t, adapter, txAdapter)
	assert.True(t, mockStore.withTxCalled)
	assert.Equal(t, mockDB, txAdapter.DB())
}

// Test interface compliance
func TestRepositoryAdapterInterfaces(t *testing.T) {
	t.Run("CardRepositoryAdapter implements CardRepository", func(t *testing.T) {
		var _ CardRepository = &cardRepositoryAdapter{}
	})

	t.Run("StatsRepositoryAdapter implements StatsRepository", func(t *testing.T) {
		var _ StatsRepository = &statsRepositoryAdapter{}
	})

	t.Run("MemoRepositoryAdapter implements MemoRepository", func(t *testing.T) {
		var _ MemoRepository = &MemoRepositoryAdapter{}
	})
}
