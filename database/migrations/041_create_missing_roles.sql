-- Migration: Create missing PostgreSQL roles (safe)
-- This migration creates the 'anon' and 'authenticated' roles only if the current user has CREATEROLE,
-- and applies grants only when the roles exist. It will not fail when the DB user lacks role creation privileges.

-- Ensure anon role exists and apply grants safely
DO $$
DECLARE
    role_exists boolean;
    can_create boolean;
BEGIN
    SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') INTO role_exists;
    SELECT COALESCE((SELECT rolcreaterole FROM pg_roles WHERE rolname = current_user), false) INTO can_create;

    IF NOT role_exists AND can_create THEN
        EXECUTE 'CREATE ROLE anon';
        role_exists := true;
    END IF;

    IF role_exists THEN
        BEGIN
            GRANT USAGE ON SCHEMA public TO anon;
            GRANT SELECT ON ALL TABLES IN SCHEMA public TO anon;
            ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO anon;
        EXCEPTION WHEN OTHERS THEN
            RAISE NOTICE 'Skipping grants for anon: %', SQLERRM;
        END;
    ELSE
        RAISE NOTICE 'Role anon not present and cannot be created by current user; skipping';
    END IF;
END
$$;

-- Ensure authenticated role exists and apply grants safely
DO $$
DECLARE
    role_exists boolean;
    can_create boolean;
BEGIN
    SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') INTO role_exists;
    SELECT COALESCE((SELECT rolcreaterole FROM pg_roles WHERE rolname = current_user), false) INTO can_create;

    IF NOT role_exists AND can_create THEN
        EXECUTE 'CREATE ROLE authenticated';
        role_exists := true;
    END IF;

    IF role_exists THEN
        BEGIN
            GRANT USAGE ON SCHEMA public TO authenticated;
            GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO authenticated;
            GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO authenticated;
            ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO authenticated;
            ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO authenticated;
        EXCEPTION WHEN OTHERS THEN
            RAISE NOTICE 'Skipping grants for authenticated: %', SQLERRM;
        END;
    ELSE
        RAISE NOTICE 'Role authenticated not present and cannot be created by current user; skipping';
    END IF;
END
$$;

-- Final pass: grant permissions for existing tables/sequences if roles exist
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
        BEGIN
            GRANT SELECT ON ALL TABLES IN SCHEMA public TO anon;
        EXCEPTION WHEN OTHERS THEN
            RAISE NOTICE 'Skipping final grants for anon: %', SQLERRM;
        END;
    END IF;

    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
        BEGIN
            GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO authenticated;
            GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO authenticated;
        EXCEPTION WHEN OTHERS THEN
            RAISE NOTICE 'Skipping final grants for authenticated: %', SQLERRM;
        END;
    END IF;
END
$$;