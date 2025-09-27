-- Migration: 001_create_execution_tables.sql
-- Description: Create initial tables for code execution tracking
-- Created: 2024

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create executions table
CREATE TABLE IF NOT EXISTS executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,

    -- Request Information
    solution_id VARCHAR(255) NOT NULL,
    challenge_id VARCHAR(255) NOT NULL,
    student_id VARCHAR(255) NOT NULL,
    language VARCHAR(50) NOT NULL,

    -- Execution Details
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    code TEXT NOT NULL,
    execution_time_ms BIGINT,
    memory_usage_mb DECIMAL(10,2),

    -- Results
    success BOOLEAN DEFAULT FALSE,
    message TEXT,
    approved_test_ids JSONB DEFAULT '[]'::jsonb,
    failed_test_ids JSONB DEFAULT '[]'::jsonb,
    total_tests INTEGER DEFAULT 0,
    passed_tests INTEGER DEFAULT 0,

    -- Error Information
    error_message TEXT,
    error_type VARCHAR(100),
    compilation_error TEXT,
    runtime_error TEXT,

    -- Metadata
    server_instance VARCHAR(255),
    client_ip VARCHAR(45),
    user_agent VARCHAR(500)
);

-- Create execution_steps table
CREATE TABLE IF NOT EXISTS execution_steps (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,

    execution_id UUID NOT NULL,
    step_name VARCHAR(100) NOT NULL,
    step_order INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms BIGINT,
    error_message TEXT,
    metadata JSONB,

    CONSTRAINT fk_execution_steps_execution_id
        FOREIGN KEY (execution_id)
        REFERENCES executions(id)
        ON DELETE CASCADE
);

-- Create execution_logs table
CREATE TABLE IF NOT EXISTS execution_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,

    execution_id UUID NOT NULL,
    level VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    source VARCHAR(100),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_execution_logs_execution_id
        FOREIGN KEY (execution_id)
        REFERENCES executions(id)
        ON DELETE CASCADE
);

-- Create test_results table
CREATE TABLE IF NOT EXISTS test_results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,

    execution_id UUID NOT NULL,
    test_id VARCHAR(255) NOT NULL,
    test_name VARCHAR(255),
    input TEXT,
    expected_output TEXT,
    actual_output TEXT,
    passed BOOLEAN DEFAULT FALSE,
    execution_time_ms BIGINT,
    error_message TEXT,

    CONSTRAINT fk_test_results_execution_id
        FOREIGN KEY (execution_id)
        REFERENCES executions(id)
        ON DELETE CASCADE
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_executions_solution_id ON executions(solution_id);
CREATE INDEX IF NOT EXISTS idx_executions_challenge_id ON executions(challenge_id);
CREATE INDEX IF NOT EXISTS idx_executions_student_id ON executions(student_id);
CREATE INDEX IF NOT EXISTS idx_executions_status ON executions(status);
CREATE INDEX IF NOT EXISTS idx_executions_created_at ON executions(created_at);
CREATE INDEX IF NOT EXISTS idx_executions_deleted_at ON executions(deleted_at);

CREATE INDEX IF NOT EXISTS idx_execution_steps_execution_id ON execution_steps(execution_id);
CREATE INDEX IF NOT EXISTS idx_execution_steps_step_order ON execution_steps(step_order);
CREATE INDEX IF NOT EXISTS idx_execution_steps_deleted_at ON execution_steps(deleted_at);

CREATE INDEX IF NOT EXISTS idx_execution_logs_execution_id ON execution_logs(execution_id);
CREATE INDEX IF NOT EXISTS idx_execution_logs_level ON execution_logs(level);
CREATE INDEX IF NOT EXISTS idx_execution_logs_timestamp ON execution_logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_execution_logs_deleted_at ON execution_logs(deleted_at);

CREATE INDEX IF NOT EXISTS idx_test_results_execution_id ON test_results(execution_id);
CREATE INDEX IF NOT EXISTS idx_test_results_test_id ON test_results(test_id);
CREATE INDEX IF NOT EXISTS idx_test_results_passed ON test_results(passed);
CREATE INDEX IF NOT EXISTS idx_test_results_deleted_at ON test_results(deleted_at);

-- Create composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_executions_student_challenge ON executions(student_id, challenge_id);
CREATE INDEX IF NOT EXISTS idx_executions_status_created ON executions(status, created_at);

-- Update trigger for updated_at columns
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply update triggers to all tables
CREATE TRIGGER update_executions_updated_at BEFORE UPDATE ON executions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_execution_steps_updated_at BEFORE UPDATE ON execution_steps
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_execution_logs_updated_at BEFORE UPDATE ON execution_logs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_test_results_updated_at BEFORE UPDATE ON test_results
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Add comments to tables
COMMENT ON TABLE executions IS 'Store code execution requests and results';
COMMENT ON TABLE execution_steps IS 'Track individual steps in the execution pipeline';
COMMENT ON TABLE execution_logs IS 'Store execution logs and debug information';
COMMENT ON TABLE test_results IS 'Store individual test case results';

-- Add comments to key columns
COMMENT ON COLUMN executions.solution_id IS 'Unique identifier for the solution being tested';
COMMENT ON COLUMN executions.challenge_id IS 'Identifier for the coding challenge';
COMMENT ON COLUMN executions.student_id IS 'Identifier for the student submitting the code';
COMMENT ON COLUMN executions.status IS 'Current execution status: pending, running, completed, failed, timed_out, cancelled';
COMMENT ON COLUMN executions.approved_test_ids IS 'JSON array of test IDs that passed';
COMMENT ON COLUMN executions.failed_test_ids IS 'JSON array of test IDs that failed';
