package http

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/luuan11/bank-system/internal/domain"
	"github.com/luuan11/bank-system/internal/usecase"
)

type stubCustomers struct {
	customer *domain.Customer
	err      error
	input    usecase.CreateCustomerInput
}

func (s *stubCustomers) create(_ context.Context, input usecase.CreateCustomerInput) (*domain.Customer, error) {
	s.input = input
	return s.customer, s.err
}

func (s *stubCustomers) get(_ context.Context, _ uuid.UUID) (*domain.Customer, error) {
	return s.customer, s.err
}

type createCustomerFunc func(context.Context, usecase.CreateCustomerInput) (*domain.Customer, error)

func (f createCustomerFunc) Execute(ctx context.Context, in usecase.CreateCustomerInput) (*domain.Customer, error) {
	return f(ctx, in)
}

type getCustomerFunc func(context.Context, uuid.UUID) (*domain.Customer, error)

func (f getCustomerFunc) Execute(ctx context.Context, id uuid.UUID) (*domain.Customer, error) {
	return f(ctx, id)
}

func customerMux(s *stubCustomers) *http.ServeMux {
	mux := http.NewServeMux()
	NewCustomer(createCustomerFunc(s.create), getCustomerFunc(s.get)).Register(mux)
	return mux
}

func TestCustomerHandlers(t *testing.T) {
	customer := &domain.Customer{
		ID:        uuid.New(),
		FullName:  "Maria Silva",
		Email:     "maria@example.com",
		CPF:       "52998224725",
		BirthDate: time.Date(1995, 4, 20, 0, 0, 0, 0, time.UTC),
	}

	t.Run("create", func(t *testing.T) {
		s := &stubCustomers{customer: customer}
		body := `{"full_name":"Maria Silva","email":"maria@example.com","cpf":"529.982.247-25","phone":"11999999999","birth_date":"1995-04-20"}`
		rec, resp := do(t, customerMux(s), http.MethodPost, "/api/customers", body)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201", rec.Code)
		}
		if s.input.CPF != "529.982.247-25" || s.input.Phone != "11999999999" || s.input.BirthDate != "1995-04-20" {
			t.Errorf("input not decoded: %+v", s.input)
		}
		data := resp.Data.(map[string]any)
		if data["birth_date"] != "1995-04-20" {
			t.Errorf("birth_date = %v, want 1995-04-20", data["birth_date"])
		}
	})

	t.Run("create conflict", func(t *testing.T) {
		s := &stubCustomers{err: domain.ErrCustomerAlreadyExists}
		rec, resp := do(t, customerMux(s), http.MethodPost, "/api/customers", `{"full_name":"x"}`)
		if rec.Code != http.StatusConflict || resp.Error.Code != "CUSTOMER_EXISTS" {
			t.Errorf("status = %d, error = %+v", rec.Code, resp.Error)
		}
	})

	t.Run("create invalid body", func(t *testing.T) {
		rec, resp := do(t, customerMux(&stubCustomers{}), http.MethodPost, "/api/customers", `not json`)
		if rec.Code != http.StatusBadRequest || resp.Error.Code != "INVALID_REQUEST" {
			t.Errorf("status = %d, error = %+v", rec.Code, resp.Error)
		}
	})

	t.Run("get", func(t *testing.T) {
		rec, resp := do(t, customerMux(&stubCustomers{customer: customer}), http.MethodGet, "/api/customers/"+customer.ID.String(), "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if resp.Data.(map[string]any)["id"] != customer.ID.String() {
			t.Errorf("unexpected data %+v", resp.Data)
		}
	})

	t.Run("get invalid id", func(t *testing.T) {
		rec, resp := do(t, customerMux(&stubCustomers{}), http.MethodGet, "/api/customers/123", "")
		if rec.Code != http.StatusBadRequest || resp.Error.Code != "INVALID_CUSTOMER_ID" {
			t.Errorf("status = %d, error = %+v", rec.Code, resp.Error)
		}
	})

	t.Run("get not found", func(t *testing.T) {
		rec, resp := do(t, customerMux(&stubCustomers{err: domain.ErrCustomerNotFound}), http.MethodGet, "/api/customers/"+uuid.NewString(), "")
		if rec.Code != http.StatusNotFound || resp.Error.Code != "CUSTOMER_NOT_FOUND" {
			t.Errorf("status = %d, error = %+v", rec.Code, resp.Error)
		}
	})
}

type stubAccounts struct {
	account     *domain.Account
	err         error
	createInput usecase.CreateAccountInput
	listInput   usecase.ListTransactionsInput
}

type createAccountFunc func(context.Context, usecase.CreateAccountInput) (*usecase.CreateAccountOutput, error)

func (f createAccountFunc) Execute(ctx context.Context, in usecase.CreateAccountInput) (*usecase.CreateAccountOutput, error) {
	return f(ctx, in)
}

type getAccountFunc func(context.Context, uuid.UUID) (*domain.Account, error)

func (f getAccountFunc) Execute(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	return f(ctx, id)
}

type listTransactionsFunc func(context.Context, usecase.ListTransactionsInput) (*usecase.ListTransactionsOutput, error)

func (f listTransactionsFunc) Execute(ctx context.Context, in usecase.ListTransactionsInput) (*usecase.ListTransactionsOutput, error) {
	return f(ctx, in)
}

func accountMux(s *stubAccounts) *http.ServeMux {
	mux := http.NewServeMux()
	NewAccount(
		createAccountFunc(func(_ context.Context, in usecase.CreateAccountInput) (*usecase.CreateAccountOutput, error) {
			s.createInput = in
			if s.err != nil {
				return nil, s.err
			}
			return &usecase.CreateAccountOutput{Account: s.account}, nil
		}),
		getAccountFunc(func(context.Context, uuid.UUID) (*domain.Account, error) {
			return s.account, s.err
		}),
		listTransactionsFunc(func(_ context.Context, in usecase.ListTransactionsInput) (*usecase.ListTransactionsOutput, error) {
			s.listInput = in
			if s.err != nil {
				return nil, s.err
			}
			return &usecase.ListTransactionsOutput{Items: []domain.Transaction{}, Limit: in.Limit, Offset: in.Offset}, nil
		}),
	).Register(mux)
	return mux
}

func TestAccountHandlers(t *testing.T) {
	account := domain.NewAccount(uuid.New(), domain.AccountTypeSavings)
	path := "/api/accounts/" + account.ID.String()

	t.Run("create", func(t *testing.T) {
		s := &stubAccounts{account: account}
		body := `{"customer_id":"` + account.CustomerID.String() + `","type":"savings"}`
		rec, _ := do(t, accountMux(s), http.MethodPost, "/api/accounts", body)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201", rec.Code)
		}
		if s.createInput.CustomerID != account.CustomerID || s.createInput.Type != domain.AccountTypeSavings {
			t.Errorf("input not decoded: %+v", s.createInput)
		}
	})

	t.Run("create errors", func(t *testing.T) {
		tests := []struct {
			err    error
			status int
			code   string
		}{
			{domain.ErrInvalidAccountType, http.StatusBadRequest, "INVALID_ACCOUNT_TYPE"},
			{domain.ErrCustomerNotFound, http.StatusNotFound, "CUSTOMER_NOT_FOUND"},
			{domain.ErrAccountAlreadyExists, http.StatusConflict, "ACCOUNT_EXISTS"},
		}
		for _, tt := range tests {
			rec, resp := do(t, accountMux(&stubAccounts{err: tt.err}), http.MethodPost, "/api/accounts", `{"type":"x"}`)
			if rec.Code != tt.status || resp.Error.Code != tt.code {
				t.Errorf("%v: status = %d, error = %+v", tt.err, rec.Code, resp.Error)
			}
		}
	})

	t.Run("create invalid customer id format", func(t *testing.T) {
		rec, resp := do(t, accountMux(&stubAccounts{}), http.MethodPost, "/api/accounts", `{"customer_id":"abc","type":"checking"}`)
		if rec.Code != http.StatusBadRequest || resp.Error.Code != "INVALID_REQUEST" {
			t.Errorf("status = %d, error = %+v", rec.Code, resp.Error)
		}
	})

	t.Run("get", func(t *testing.T) {
		rec, resp := do(t, accountMux(&stubAccounts{account: account}), http.MethodGet, path, "")
		if rec.Code != http.StatusOK || resp.Data.(map[string]any)["type"] != "savings" {
			t.Errorf("status = %d, data = %+v", rec.Code, resp.Data)
		}
	})

	t.Run("get invalid id", func(t *testing.T) {
		rec, resp := do(t, accountMux(&stubAccounts{}), http.MethodGet, "/api/accounts/nope", "")
		if rec.Code != http.StatusBadRequest || resp.Error.Code != "INVALID_ACCOUNT_ID" {
			t.Errorf("status = %d, error = %+v", rec.Code, resp.Error)
		}
	})

	t.Run("get not found", func(t *testing.T) {
		rec, resp := do(t, accountMux(&stubAccounts{err: domain.ErrAccountNotFound}), http.MethodGet, path, "")
		if rec.Code != http.StatusNotFound || resp.Error.Code != "ACCOUNT_NOT_FOUND" {
			t.Errorf("status = %d, error = %+v", rec.Code, resp.Error)
		}
	})

	t.Run("list transactions", func(t *testing.T) {
		s := &stubAccounts{}
		rec, resp := do(t, accountMux(s), http.MethodGet, path+"/transactions?limit=5&offset=10", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if s.listInput.AccountID != account.ID || s.listInput.Limit != 5 || s.listInput.Offset != 10 {
			t.Errorf("input = %+v", s.listInput)
		}
		if items := resp.Data.(map[string]any)["items"]; items == nil {
			t.Error("items must be an empty array, not null")
		}
	})

	t.Run("list transactions defaults", func(t *testing.T) {
		s := &stubAccounts{}
		do(t, accountMux(s), http.MethodGet, path+"/transactions", "")
		if s.listInput.Limit != 0 || s.listInput.Offset != 0 {
			t.Errorf("missing params should be zero (use case applies defaults), got %+v", s.listInput)
		}
	})

	t.Run("list transactions bad params", func(t *testing.T) {
		tests := map[string]string{
			path + "/transactions?limit=ten":  "INVALID_LIMIT",
			path + "/transactions?offset=1.5": "INVALID_OFFSET",
			"/api/accounts/x/transactions":    "INVALID_ACCOUNT_ID",
		}
		for p, code := range tests {
			rec, resp := do(t, accountMux(&stubAccounts{}), http.MethodGet, p, "")
			if rec.Code != http.StatusBadRequest || resp.Error.Code != code {
				t.Errorf("%s: status = %d, error = %+v", p, rec.Code, resp.Error)
			}
		}
	})

	t.Run("list transactions account not found", func(t *testing.T) {
		rec, _ := do(t, accountMux(&stubAccounts{err: domain.ErrAccountNotFound}), http.MethodGet, path+"/transactions", "")
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
}

type pingFunc func(context.Context) error

func (f pingFunc) PingContext(ctx context.Context) error { return f(ctx) }

func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
	}{
		{"healthy", nil, http.StatusOK},
		{"database down", errors.New("connection refused"), http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			NewHealth(pingFunc(func(ctx context.Context) error {
				if _, ok := ctx.Deadline(); !ok {
					t.Error("ping must have a timeout")
				}
				return tt.err
			})).Register(mux)

			rec, resp := do(t, mux, http.MethodGet, "/health", "")
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.status)
			}
			if tt.err != nil && resp.Error.Code != "DATABASE_UNAVAILABLE" {
				t.Errorf("error = %+v", resp.Error)
			}
		})
	}
}
