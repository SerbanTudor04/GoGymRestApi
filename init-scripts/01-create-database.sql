CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Set timezone
SET timezone = 'UTC';

-- Create custom types if needed
DO $$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_status') THEN
            CREATE TYPE user_status AS ENUM ('active', 'inactive', 'suspended');
        END IF;

        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'membership_status') THEN
            CREATE TYPE membership_status AS ENUM ('active', 'expired', 'suspended');
        END IF;

        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'client_pass_action') THEN
            CREATE TYPE client_pass_action AS ENUM ('in', 'out');
        END IF;
    END
$$;