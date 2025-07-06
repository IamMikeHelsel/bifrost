'''use thiserror::Error;

#[derive(Error, Debug)]
pub enum ModbusError {
    #[error("I/O error: {0}")]
    Io(#[from] std::io::Error),

    #[error("Invalid data")]
    InvalidData,

    #[error("Invalid function code: {0}")]
    InvalidFunctionCode(u8),
}
''