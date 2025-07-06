use thiserror::Error;

#[derive(Error, Debug)]
pub enum ModbusError {
    #[error("Invalid function code: {0}")]
    InvalidFunctionCode(u8),
    
    #[error("Invalid data address: {0}")]
    InvalidDataAddress(u16),
    
    #[error("Invalid data value: {0}")]
    InvalidDataValue(u16),
    
    #[error("Device failure")]
    DeviceFailure,
    
    #[error("CRC error")]
    CrcError,
    
    #[error("Frame too short: expected at least {expected} bytes, got {actual}")]
    FrameTooShort { expected: usize, actual: usize },
    
    #[error("Invalid frame format")]
    InvalidFrame,
    
    #[error("Buffer overflow")]
    BufferOverflow,
    
    #[error("Timeout")]
    Timeout,
    
    #[error("IO error: {0}")]
    Io(#[from] std::io::Error),
}

impl ModbusError {
    pub fn to_exception_code(&self) -> u8 {
        match self {
            ModbusError::InvalidFunctionCode(_) => 0x01,
            ModbusError::InvalidDataAddress(_) => 0x02,
            ModbusError::InvalidDataValue(_) => 0x03,
            ModbusError::DeviceFailure => 0x04,
            _ => 0x04, // Generic device failure for other errors
        }
    }
}