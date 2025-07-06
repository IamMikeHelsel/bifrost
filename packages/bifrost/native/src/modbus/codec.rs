'''use bytes::{Buf, BufMut, BytesMut};
use tokio_util::codec::{Decoder, Encoder};

use crate::modbus::error::ModbusError;
use crate::modbus::frame::{Header, Request, Response};

pub struct ModbusCodec;

impl Encoder<Request> for ModbusCodec {
    type Error = ModbusError;

    fn encode(&mut self, item: Request, dst: &mut BytesMut) -> Result<(), Self::Error> {
        dst.put_u16(item.header.transaction_id);
        dst.put_u16(item.header.protocol_id);
        dst.put_u16(item.header.length);
        dst.put_u8(item.header.unit_id);
        dst.put_u8(item.function as u8);
        dst.put_u16(item.address);
        dst.put_u16(item.count);
        Ok(())
    }
}

impl Decoder for ModbusCodec {
    type Item = Response;
    type Error = ModbusError;

    fn decode(&mut self, src: &mut BytesMut) -> Result<Option<Self::Item>, Self::Error> {
        if src.len() < 8 {
            return Ok(None);
        }

        let transaction_id = src.get_u16();
        let protocol_id = src.get_u16();
        let length = src.get_u16();
        let unit_id = src.get_u8();

        if src.len() < length as usize {
            return Ok(None);
        }

        let function_code = src.get_u8();

        let data = src.split_to(length as usize - 2).to_vec();

        Ok(Some(Response {
            header: Header {
                transaction_id,
                protocol_id,
                length,
                unit_id,
            },
            function: match function_code {
                0x03 => crate::modbus::frame::FunctionCode::ReadHoldingRegisters,
                0x06 => crate::modbus::frame::FunctionCode::WriteSingleRegister,
                _ => return Err(ModbusError::InvalidFunctionCode(function_code)),
            },
            data,
        }))
    }
}
'''