use bytes::{Bytes, BytesMut};
use std::fmt;

#[repr(u8)]
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum FunctionCode {
    ReadCoils = 0x01,
    ReadDiscreteInputs = 0x02,
    ReadHoldingRegisters = 0x03,
    ReadInputRegisters = 0x04,
    WriteSingleCoil = 0x05,
    WriteSingleRegister = 0x06,
    WriteMultipleCoils = 0x0F,
    WriteMultipleRegisters = 0x10,
}

impl FunctionCode {
    pub fn from_u8(value: u8) -> Option<Self> {
        match value {
            0x01 => Some(FunctionCode::ReadCoils),
            0x02 => Some(FunctionCode::ReadDiscreteInputs),
            0x03 => Some(FunctionCode::ReadHoldingRegisters),
            0x04 => Some(FunctionCode::ReadInputRegisters),
            0x05 => Some(FunctionCode::WriteSingleCoil),
            0x06 => Some(FunctionCode::WriteSingleRegister),
            0x0F => Some(FunctionCode::WriteMultipleCoils),
            0x10 => Some(FunctionCode::WriteMultipleRegisters),
            _ => None,
        }
    }
}

#[derive(Debug, Clone)]
pub struct ModbusFrame {
    pub unit_id: u8,
    pub function_code: FunctionCode,
    pub data: Bytes,
}

impl ModbusFrame {
    pub fn new(unit_id: u8, function_code: FunctionCode, data: Bytes) -> Self {
        ModbusFrame {
            unit_id,
            function_code,
            data,
        }
    }
    
    pub fn to_bytes(&self) -> BytesMut {
        let mut bytes = BytesMut::with_capacity(self.data.len() + 2);
        bytes.extend_from_slice(&[self.unit_id, self.function_code as u8]);
        bytes.extend_from_slice(&self.data);
        bytes
    }
}

#[derive(Debug, Clone)]
pub enum ModbusRequest {
    ReadCoils {
        address: u16,
        quantity: u16,
    },
    ReadDiscreteInputs {
        address: u16,
        quantity: u16,
    },
    ReadHoldingRegisters {
        address: u16,
        quantity: u16,
    },
    ReadInputRegisters {
        address: u16,
        quantity: u16,
    },
    WriteSingleCoil {
        address: u16,
        value: bool,
    },
    WriteSingleRegister {
        address: u16,
        value: u16,
    },
    WriteMultipleCoils {
        address: u16,
        values: Vec<bool>,
    },
    WriteMultipleRegisters {
        address: u16,
        values: Vec<u16>,
    },
}

impl ModbusRequest {
    pub fn to_frame(&self, unit_id: u8) -> ModbusFrame {
        let (function_code, data) = match self {
            ModbusRequest::ReadCoils { address, quantity } |
            ModbusRequest::ReadDiscreteInputs { address, quantity } |
            ModbusRequest::ReadHoldingRegisters { address, quantity } |
            ModbusRequest::ReadInputRegisters { address, quantity } => {
                let fc = match self {
                    ModbusRequest::ReadCoils { .. } => FunctionCode::ReadCoils,
                    ModbusRequest::ReadDiscreteInputs { .. } => FunctionCode::ReadDiscreteInputs,
                    ModbusRequest::ReadHoldingRegisters { .. } => FunctionCode::ReadHoldingRegisters,
                    ModbusRequest::ReadInputRegisters { .. } => FunctionCode::ReadInputRegisters,
                    _ => unreachable!(),
                };
                let mut data = BytesMut::with_capacity(4);
                data.extend_from_slice(&address.to_be_bytes());
                data.extend_from_slice(&quantity.to_be_bytes());
                (fc, data.freeze())
            }
            ModbusRequest::WriteSingleCoil { address, value } => {
                let mut data = BytesMut::with_capacity(4);
                data.extend_from_slice(&address.to_be_bytes());
                data.extend_from_slice(&if *value { 0xFF00u16 } else { 0x0000u16 }.to_be_bytes());
                (FunctionCode::WriteSingleCoil, data.freeze())
            }
            ModbusRequest::WriteSingleRegister { address, value } => {
                let mut data = BytesMut::with_capacity(4);
                data.extend_from_slice(&address.to_be_bytes());
                data.extend_from_slice(&value.to_be_bytes());
                (FunctionCode::WriteSingleRegister, data.freeze())
            }
            ModbusRequest::WriteMultipleCoils { address, values } => {
                let byte_count = (values.len() + 7) / 8;
                let mut data = BytesMut::with_capacity(5 + byte_count);
                data.extend_from_slice(&address.to_be_bytes());
                data.extend_from_slice(&(values.len() as u16).to_be_bytes());
                data.extend_from_slice(&[byte_count as u8]);
                
                // Pack coils into bytes
                let mut packed = vec![0u8; byte_count];
                for (i, &value) in values.iter().enumerate() {
                    if value {
                        packed[i / 8] |= 1 << (i % 8);
                    }
                }
                data.extend_from_slice(&packed);
                (FunctionCode::WriteMultipleCoils, data.freeze())
            }
            ModbusRequest::WriteMultipleRegisters { address, values } => {
                let byte_count = values.len() * 2;
                let mut data = BytesMut::with_capacity(5 + byte_count);
                data.extend_from_slice(&address.to_be_bytes());
                data.extend_from_slice(&(values.len() as u16).to_be_bytes());
                data.extend_from_slice(&[byte_count as u8]);
                
                for value in values {
                    data.extend_from_slice(&value.to_be_bytes());
                }
                (FunctionCode::WriteMultipleRegisters, data.freeze())
            }
        };
        
        ModbusFrame::new(unit_id, function_code, data)
    }
}

#[derive(Debug, Clone)]
pub enum ModbusResponse {
    ReadCoils(Vec<bool>),
    ReadDiscreteInputs(Vec<bool>),
    ReadHoldingRegisters(Vec<u16>),
    ReadInputRegisters(Vec<u16>),
    WriteSingleCoil { address: u16, value: bool },
    WriteSingleRegister { address: u16, value: u16 },
    WriteMultipleCoils { address: u16, quantity: u16 },
    WriteMultipleRegisters { address: u16, quantity: u16 },
    Exception { function: u8, exception_code: u8 },
}

impl fmt::Display for ModbusResponse {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            ModbusResponse::ReadCoils(values) => write!(f, "ReadCoils: {} values", values.len()),
            ModbusResponse::ReadDiscreteInputs(values) => write!(f, "ReadDiscreteInputs: {} values", values.len()),
            ModbusResponse::ReadHoldingRegisters(values) => write!(f, "ReadHoldingRegisters: {} values", values.len()),
            ModbusResponse::ReadInputRegisters(values) => write!(f, "ReadInputRegisters: {} values", values.len()),
            ModbusResponse::WriteSingleCoil { address, value } => write!(f, "WriteSingleCoil: address={}, value={}", address, value),
            ModbusResponse::WriteSingleRegister { address, value } => write!(f, "WriteSingleRegister: address={}, value={}", address, value),
            ModbusResponse::WriteMultipleCoils { address, quantity } => write!(f, "WriteMultipleCoils: address={}, quantity={}", address, quantity),
            ModbusResponse::WriteMultipleRegisters { address, quantity } => write!(f, "WriteMultipleRegisters: address={}, quantity={}", address, quantity),
            ModbusResponse::Exception { function, exception_code } => write!(f, "Exception: function={}, code={}", function, exception_code),
        }
    }
}