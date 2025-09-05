-- services/staff/cmd/migrate/migrations/20250905204930_create-drivers.up.sql
CREATE TABLE IF NOT EXISTS drivers (
    internal_id BIGINT UNSIGNED PRIMARY KEY,
    external_id BINARY(16) UNIQUE NOT NULL,
    user_id VARCHAR(36) UNIQUE NOT NULL,
    license_number VARCHAR(50) UNIQUE NOT NULL,
    license_class ENUM('LICENSE_UNSPECIFIED', 'CLASS_A', 'CLASS_B', 'CLASS_C', 'CLASS_D', 'CLASS_E') NOT NULL DEFAULT 'CLASS_B',
    license_expiry DATE NOT NULL,
    experience_years INT NOT NULL DEFAULT 0,
    phone_number VARCHAR(20) NOT NULL,
    emergency_contact_name VARCHAR(100) NOT NULL,
    emergency_contact_phone VARCHAR(20) NOT NULL,
    status ENUM('STATUS_UNSPECIFIED', 'PENDING_VERIFICATION', 'ACTIVE', 'SUSPENDED', 'INACTIVE') NOT NULL DEFAULT 'PENDING_VERIFICATION',
    hire_date DATE NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP(6),
    
    INDEX idx_drivers_user_id (user_id),
    INDEX idx_drivers_license (license_number),
    INDEX idx_drivers_status (status),
    INDEX idx_drivers_license_class (license_class),
    INDEX idx_drivers_license_expiry (license_expiry),
    INDEX idx_drivers_created_at (created_at)
);