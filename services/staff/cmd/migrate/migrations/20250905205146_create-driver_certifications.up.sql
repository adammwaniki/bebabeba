-- services/staff/cmd/migrate/migrations/20250905205146_create-driver_certifications.up.sql
CREATE TABLE IF NOT EXISTS driver_certifications (
    id BIGINT UNSIGNED PRIMARY KEY,
    driver_id BINARY(16) NOT NULL,
    certification_name VARCHAR(100) NOT NULL,
    issued_by VARCHAR(100) NOT NULL,
    issue_date DATE NOT NULL,
    expiry_date DATE NOT NULL,
    status ENUM('CERT_STATUS_UNSPECIFIED', 'CERT_ACTIVE', 'CERT_EXPIRED', 'CERT_SUSPENDED', 'CERT_REVOKED') NOT NULL DEFAULT 'CERT_ACTIVE',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP(6),
    
    INDEX idx_certifications_driver (driver_id),
    INDEX idx_certifications_status (status),
    INDEX idx_certifications_expiry (expiry_date),
    INDEX idx_certifications_name (certification_name),
    
    CONSTRAINT fk_certifications_driver 
        FOREIGN KEY (driver_id) REFERENCES drivers(external_id) 
        ON DELETE CASCADE
);