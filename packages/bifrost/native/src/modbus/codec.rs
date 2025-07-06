use bytes::{Buf, BufMut, Bytes, BytesMut};
use crc16::*;
use byteorder::{BigEndian, ByteOrder};
use std::io::Cursor;

use super::error::ModbusError;
use super::frame::{FunctionCode, ModbusFrame, ModbusResponse};

const MIN_FRAME_SIZE: usize = 4; // Unit ID + Function Code + 2 bytes CRC
const MAX_FRAME_SIZE: usize = 260; // Max ADU size for Modbus RTU

pub struct ModbusEncoder;

impl ModbusEncoder {
    pub fn encode_rtu(frame: &ModbusFrame) -> Result<BytesMut, ModbusError> {
        let mut buf = frame.to_bytes();
        
        // Calculate and append CRC16
        let crc = State::<MODBUS>::calculate(&buf);
        buf.put_u16_le(crc);
        
        Ok(buf)
    }
    
    pub fn encode_tcp(frame: &ModbusFrame, transaction_id: u16) -> Result<BytesMut, ModbusError> {
        let pdu_len = 1 + 1 + frame.data.len(); // unit_id + function_code + data
        let mut buf = BytesMut::with_capacity(7 + pdu_len);
        
        // MBAP Header
        buf.put_u16(transaction_id);
        buf.put_u16(0); // Protocol ID (always 0 for Modbus)
        buf.put_u16(pdu_len as u16);
        
        // PDU
        buf.extend_from_slice(&frame.to_bytes());
        
        Ok(buf)
    }
}

pub struct ModbusDecoder;

impl ModbusDecoder {
    pub fn decode_rtu(data: &[u8]) -> Result<ModbusFrame, ModbusError> {
        if data.len() < MIN_FRAME_SIZE {
            return Err(ModbusError::FrameTooShort {
                expected: MIN_FRAME_SIZE,
                actual: data.len(),
            });
        }
        
        // Verify CRC
        let frame_data = &data[..data.len() - 2];
        let received_crc = u16::from_le_bytes([data[data.len() - 2], data[data.len() - 1]]);
        let calculated_crc = State::<MODBUS>::calculate(frame_data);
        
        if received_crc != calculated_crc {
            return Err(ModbusError::CrcError);
        }
        
        // Parse frame
        let unit_id = frame_data[0];
        let function_code = FunctionCode::from_u8(frame_data[1])
            .ok_or_else(|| ModbusError::InvalidFunctionCode(frame_data[1]))?;
        let data = Bytes::copy_from_slice(&frame_data[2..]);
        
        Ok(ModbusFrame::new(unit_id, function_code, data))
    }
    
    pub fn decode_tcp(data: &[u8]) -> Result<(u16, ModbusFrame), ModbusError> {
        if data.len() < 7 {
            return Err(ModbusError::FrameTooShort {
                expected: 7,
                actual: data.len(),
            });
        }
        
        // Read MBAP header
        let transaction_id = BigEndian::read_u16(&data[0..2]);
        let protocol_id = BigEndian::read_u16(&data[2..4]);
        let length = BigEndian::read_u16(&data[4..6]) as usize;
        
        if protocol_id != 0 {
            return Err(ModbusError::InvalidFrame);
        }
        
        if data.len() < 6 + length {
            return Err(ModbusError::FrameTooShort {
                expected: 6 + length,
                actual: data.len(),
            });
        }
        
        // Parse PDU
        let unit_id = data[6];
        let function_code = FunctionCode::from_u8(data[7])
            .ok_or_else(|| ModbusError::InvalidFunctionCode(data[7]))?;
        
        let data_start = 8;
        let data_end = 6 + length;
        let frame_data = Bytes::copy_from_slice(&data[data_start..data_end]);
        
        Ok((transaction_id, ModbusFrame::new(unit_id, function_code, frame_data)))
    }
    
    pub fn decode_response(frame: &ModbusFrame, request_function: FunctionCode) -> Result<ModbusResponse, ModbusError> {
        // Check for exception response
        if frame.function_code as u8 & 0x80 != 0 {
            let exception_code = frame.data.get(0).copied().unwrap_or(0);
            return Ok(ModbusResponse::Exception {
                function: request_function as u8,
                exception_code,
            });
        }
        
        let data = frame.data.as_ref();
        
        match frame.function_code {
            FunctionCode::ReadCoils | FunctionCode::ReadDiscreteInputs => {
                if data.is_empty() {
                    return Err(ModbusError::InvalidFrame);
                }
                let byte_count = data[0] as usize;
                
                if data.len() < 1 + byte_count {
                    return Err(ModbusError::InvalidFrame);
                }
                
                let mut coils = Vec::new();
                for i in 0..byte_count {
                    let byte = data[1 + i];
                    for bit in 0..8 {
                        if i * 8 + bit < byte_count * 8 {
                            coils.push((byte >> bit) & 1 != 0);
                        }
                    }
                }
                
                match frame.function_code {
                    FunctionCode::ReadCoils => Ok(ModbusResponse::ReadCoils(coils)),
                    FunctionCode::ReadDiscreteInputs => Ok(ModbusResponse::ReadDiscreteInputs(coils)),
                    _ => unreachable!(),
                }
            }
            
            FunctionCode::ReadHoldingRegisters | FunctionCode::ReadInputRegisters => {
                let byte_count = cursor.read_u8()
                    .map_err(|_| ModbusError::InvalidFrame)? as usize;
                
                if byte_count % 2 != 0 || cursor.get_ref().len() < 1 + byte_count {
                    return Err(ModbusError::InvalidFrame);
                }
                
                let register_count = byte_count / 2;
                let mut registers = Vec::with_capacity(register_count);
                
                for _ in 0..register_count {
                    let value = cursor.read_u16::<BigEndian>().unwrap();
                    registers.push(value);
                }
                
                match frame.function_code {
                    FunctionCode::ReadHoldingRegisters => Ok(ModbusResponse::ReadHoldingRegisters(registers)),
                    FunctionCode::ReadInputRegisters => Ok(ModbusResponse::ReadInputRegisters(registers)),
                    _ => unreachable!(),
                }
            }
            
            FunctionCode::WriteSingleCoil => {
                let address = cursor.read_u16::<BigEndian>()
                    .map_err(|_| ModbusError::InvalidFrame)?;
                let value = cursor.read_u16::<BigEndian>()
                    .map_err(|_| ModbusError::InvalidFrame)?;
                
                Ok(ModbusResponse::WriteSingleCoil {
                    address,
                    value: value == 0xFF00,
                })
            }
            
            FunctionCode::WriteSingleRegister => {
                let address = cursor.read_u16::<BigEndian>()
                    .map_err(|_| ModbusError::InvalidFrame)?;
                let value = cursor.read_u16::<BigEndian>()
                    .map_err(|_| ModbusError::InvalidFrame)?;
                
                Ok(ModbusResponse::WriteSingleRegister { address, value })
            }
            
            FunctionCode::WriteMultipleCoils => {
                let address = cursor.read_u16::<BigEndian>()
                    .map_err(|_| ModbusError::InvalidFrame)?;
                let quantity = cursor.read_u16::<BigEndian>()
                    .map_err(|_| ModbusError::InvalidFrame)?;
                
                Ok(ModbusResponse::WriteMultipleCoils { address, quantity })
            }
            
            FunctionCode::WriteMultipleRegisters => {
                let address = cursor.read_u16::<BigEndian>()
                    .map_err(|_| ModbusError::InvalidFrame)?;
                let quantity = cursor.read_u16::<BigEndian>()
                    .map_err(|_| ModbusError::InvalidFrame)?;
                
                Ok(ModbusResponse::WriteMultipleRegisters { address, quantity })
            }
        }
    }
}