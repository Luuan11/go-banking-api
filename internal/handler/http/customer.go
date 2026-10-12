package http

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/luuan11/bank-system/internal/domain"
	"github.com/luuan11/bank-system/internal/usecase"
)

type customerCreator interface {
	Execute(ctx context.Context, input usecase.CreateCustomerInput) (*domain.Customer, error)
}

type customerGetter interface {
	Execute(ctx context.Context, id uuid.UUID) (*domain.Customer, error)
}

type Customer struct {
	createCustomerUC customerCreator
	getCustomerUC    customerGetter
}

func NewCustomer(createCustomerUC customerCreator, getCustomerUC customerGetter) *Customer {
	return &Customer{
		createCustomerUC: createCustomerUC,
		getCustomerUC:    getCustomerUC,
	}
}

func (h *Customer) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/customers", h.Create)
	mux.HandleFunc("GET /api/customers/{id}", h.Get)
}

type customerResponse struct {
	ID        uuid.UUID `json:"id"`
	FullName  string    `json:"full_name"`
	Email     string    `json:"email"`
	CPF       string    `json:"cpf"`
	Phone     string    `json:"phone,omitempty"`
	BirthDate string    `json:"birth_date"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toCustomerResponse(c *domain.Customer) customerResponse {
	return customerResponse{
		ID:        c.ID,
		FullName:  c.FullName,
		Email:     c.Email,
		CPF:       c.CPF,
		Phone:     c.Phone,
		BirthDate: c.BirthDate.Format(time.DateOnly),
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func (h *Customer) Create(w http.ResponseWriter, r *http.Request) {
	var input usecase.CreateCustomerInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, r, err)
		return
	}

	customer, err := h.createCustomerUC.Execute(r.Context(), input)
	if err != nil {
		writeError(w, r, err)
		return
	}

	JSON(w, http.StatusCreated, toCustomerResponse(customer))
}

func (h *Customer) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, r, domain.ErrInvalidCustomerID)
		return
	}

	customer, err := h.getCustomerUC.Execute(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	JSON(w, http.StatusOK, toCustomerResponse(customer))
}
