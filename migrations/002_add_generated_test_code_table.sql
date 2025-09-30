-- Migration: 002_add_generated_test_code_table.sql
-- Description: Add table to store generated test code for auditing and debugging
-- Created: 2025

-- Create generated_test_code table
CREATE TABLE IF NOT EXISTS generated_test_code (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,

    -- Reference to execution
    execution_id UUID NOT NULL,

    -- Test code information
    language VARCHAR(50) NOT NULL,
    generator_type VARCHAR(50) NOT NULL, -- e.g., 'cpp_generator', 'python_generator'
    test_code TEXT NOT NULL,

    -- Metadata
    challenge_id VARCHAR(255),
    test_cases_count INTEGER DEFAULT 0,
    has_custom_validation BOOLEAN DEFAULT FALSE,

    -- Performance metrics
    generation_time_ms BIGINT,
    code_size_bytes INTEGER,

    CONSTRAINT fk_generated_test_code_execution_id
        FOREIGN KEY (execution_id)
        REFERENCES executions(id)
        ON DELETE CASCADE
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_generated_test_code_execution_id ON generated_test_code(execution_id);
CREATE INDEX IF NOT EXISTS idx_generated_test_code_language ON generated_test_code(language);
CREATE INDEX IF NOT EXISTS idx_generated_test_code_generator_type ON generated_test_code(generator_type);
CREATE INDEX IF NOT EXISTS idx_generated_test_code_challenge_id ON generated_test_code(challenge_id);
CREATE INDEX IF NOT EXISTS idx_generated_test_code_created_at ON generated_test_code(created_at);
CREATE INDEX IF NOT EXISTS idx_generated_test_code_deleted_at ON generated_test_code(deleted_at);

-- Update trigger for updated_at column
CREATE TRIGGER update_generated_test_code_updated_at BEFORE UPDATE ON generated_test_code
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Add comments to table and columns
COMMENT ON TABLE generated_test_code IS 'Store generated test code for each execution for auditing and debugging purposes';
COMMENT ON COLUMN generated_test_code.execution_id IS 'Reference to the execution that generated this test code';
COMMENT ON COLUMN generated_test_code.language IS 'Programming language of the generated test code';
COMMENT ON COLUMN generated_test_code.generator_type IS 'Type of code generator used (e.g., cpp_generator, python_generator)';
COMMENT ON COLUMN generated_test_code.test_code IS 'The complete generated test code including solution and test cases';
COMMENT ON COLUMN generated_test_code.challenge_id IS 'Challenge identifier for grouping and analysis';
COMMENT ON COLUMN generated_test_code.test_cases_count IS 'Number of test cases included in the generated code';
COMMENT ON COLUMN generated_test_code.has_custom_validation IS 'Whether the test code includes custom validation logic';
COMMENT ON COLUMN generated_test_code.generation_time_ms IS 'Time taken to generate the test code in milliseconds';
COMMENT ON COLUMN generated_test_code.code_size_bytes IS 'Size of the generated test code in bytes';