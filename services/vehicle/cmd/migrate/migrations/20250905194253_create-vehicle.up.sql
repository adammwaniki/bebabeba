-- services/vehicle/cmd/migrate/migrations/20250905194253_create-vehicle.up.sql
CREATE TABLE IF NOT EXISTS vehicles (
    internal_id BIGINT UNSIGNED PRIMARY KEY,
    external_id BINARY(16) UNIQUE NOT NULL,
    vehicle_type_id INT NOT NULL,
    license_plate VARCHAR(20) UNIQUE NOT NULL,
    make VARCHAR(50) NOT NULL,
    model VARCHAR(50) NOT NULL,
    year INT NOT NULL,
    color VARCHAR(30) NOT NULL,
    seating_capacity INT NOT NULL,
    fuel_type ENUM('FUEL_UNSPECIFIED', 'PETROL', 'DIESEL', 'ELECTRIC', 'HYBRID') NOT NULL DEFAULT 'PETROL',
    engine_number VARCHAR(100) NULL,
    chassis_number VARCHAR(100) NULL,
    registration_date DATE NULL,
    insurance_expiry DATE NULL,
    status ENUM('STATUS_UNSPECIFIED', 'ACTIVE', 'MAINTENANCE', 'RETIRED', 'ASSIGNED') NOT NULL DEFAULT 'ACTIVE',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP(6),
    
    FOREIGN KEY (vehicle_type_id) REFERENCES vehicle_types(id) ON DELETE RESTRICT,
    INDEX idx_vehicles_type (vehicle_type_id),
    INDEX idx_vehicles_status (status),
    INDEX idx_vehicles_license (license_plate),
    INDEX idx_vehicles_make (make),
    INDEX idx_vehicles_created_at (created_at),
    INDEX idx_vehicles_insurance_expiry (insurance_expiry)
);