-- services/vehicle/cmd/migrate/migrations/20250905191145_create-vehicle_types.up.sql
CREATE TABLE IF NOT EXISTS vehicle_types (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    INDEX idx_vehicle_types_name (name)
);

-- Insert standard vehicle types
INSERT IGNORE INTO vehicle_types (name, description) VALUES 
('cab', 'Taxi cabs for individual passenger transport'),
('bus', 'Large passenger buses for city-to-city routes'),
('matatu', 'Shared taxis for local and regional routes'),
('bodaboda', 'Motorcycle taxis for short distance transport'),
('truck', 'Cargo vehicles for goods transport'),
('van', 'Small passenger or cargo vans'),
('pickup', 'Pickup trucks for light cargo transport');