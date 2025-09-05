-- services/staff/cmd/migrate/migrations/20250905205338_create-driver_status_history.up.sql
-- Driver status history table (for audit trail)
CREATE TABLE IF NOT EXISTS driver_status_history (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    driver_id BINARY(16) NOT NULL,
    previous_status ENUM('STATUS_UNSPECIFIED', 'PENDING_VERIFICATION', 'ACTIVE', 'SUSPENDED', 'INACTIVE') NOT NULL,
    new_status ENUM('STATUS_UNSPECIFIED', 'PENDING_VERIFICATION', 'ACTIVE', 'SUSPENDED', 'INACTIVE') NOT NULL,
    reason TEXT,
    changed_by VARCHAR(36), -- User ID who made the change
    changed_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    
    INDEX idx_status_history_driver (driver_id),
    INDEX idx_status_history_date (changed_at),
    
    CONSTRAINT fk_status_history_driver 
        FOREIGN KEY (driver_id) REFERENCES drivers(external_id) 
        ON DELETE CASCADE
);