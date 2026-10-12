CREATE TABLE customers (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	full_name VARCHAR(255) NOT NULL,
	email VARCHAR(255) NOT NULL,
	cpf CHAR(11) NOT NULL,
	phone VARCHAR(20),
	birth_date DATE NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	CONSTRAINT customers_email_key UNIQUE (email),
	CONSTRAINT customers_cpf_key UNIQUE (cpf)
);

CREATE TABLE accounts (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
	number VARCHAR(20) NOT NULL,
	type VARCHAR(20) NOT NULL,
	balance BIGINT NOT NULL DEFAULT 0,
	daily_limit BIGINT NOT NULL DEFAULT 100000,
	is_active BOOLEAN NOT NULL DEFAULT TRUE,
	blocked_at TIMESTAMPTZ,
	blocked_reason TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	CONSTRAINT accounts_number_key UNIQUE (number),
	CONSTRAINT accounts_customer_type_key UNIQUE (customer_id, type),
	CONSTRAINT accounts_type_check CHECK (type IN ('checking', 'savings')),
	CONSTRAINT accounts_balance_check CHECK (balance >= 0),
	CONSTRAINT accounts_daily_limit_check CHECK (daily_limit >= 0)
);

CREATE TABLE transactions (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
	transaction_type VARCHAR(20) NOT NULL,
	amount BIGINT NOT NULL,
	balance_before BIGINT NOT NULL,
	balance_after BIGINT NOT NULL,
	description TEXT,
	reference_id UUID,
	metadata JSONB,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	CONSTRAINT transactions_type_check CHECK (
		transaction_type IN ('deposit', 'withdrawal', 'transfer_in', 'transfer_out', 'payment', 'refund')
	),
	CONSTRAINT transactions_amount_check CHECK (amount > 0)
);

CREATE INDEX idx_transactions_account_created_at ON transactions (account_id, created_at DESC);
CREATE INDEX idx_transactions_reference_id ON transactions (reference_id) WHERE reference_id IS NOT NULL;
