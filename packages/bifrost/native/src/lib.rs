mod modbus;

use pyo3::prelude::*;
use pyo3::types::PyBytes;
use bytes::Bytes;

use modbus::{ModbusEncoder, ModbusDecoder, ModbusFrame, FunctionCode, ModbusRequest, ModbusResponse};

/// Encode a Modbus RTU frame
#[pyfunction]
fn encode_rtu_frame(unit_id: u8, function_code: u8, data: &[u8]) -> PyResult<Vec<u8>> {
    let fc = FunctionCode::from_u8(function_code)
        .ok_or_else(|| PyErr::new::<pyo3::exceptions::PyValueError, _>(
            format!("Invalid function code: {}", function_code)
        ))?;
    
    let frame = ModbusFrame::new(unit_id, fc, Bytes::copy_from_slice(data));
    let encoded = ModbusEncoder::encode_rtu(&frame)
        .map_err(|e| PyErr::new::<pyo3::exceptions::PyRuntimeError, _>(e.to_string()))?;
    
    Ok(encoded.to_vec())
}

/// Decode a Modbus RTU frame
#[pyfunction]
fn decode_rtu_frame(data: &[u8]) -> PyResult<(u8, u8, Vec<u8>)> {
    let frame = ModbusDecoder::decode_rtu(data)
        .map_err(|e| PyErr::new::<pyo3::exceptions::PyRuntimeError, _>(e.to_string()))?;
    
    Ok((frame.unit_id, frame.function_code as u8, frame.data.to_vec()))
}

/// Encode a Modbus TCP frame
#[pyfunction]
fn encode_tcp_frame(transaction_id: u16, unit_id: u8, function_code: u8, data: &[u8]) -> PyResult<Vec<u8>> {
    let fc = FunctionCode::from_u8(function_code)
        .ok_or_else(|| PyErr::new::<pyo3::exceptions::PyValueError, _>(
            format!("Invalid function code: {}", function_code)
        ))?;
    
    let frame = ModbusFrame::new(unit_id, fc, Bytes::copy_from_slice(data));
    let encoded = ModbusEncoder::encode_tcp(&frame, transaction_id)
        .map_err(|e| PyErr::new::<pyo3::exceptions::PyRuntimeError, _>(e.to_string()))?;
    
    Ok(encoded.to_vec())
}

/// Decode a Modbus TCP frame
#[pyfunction]
fn decode_tcp_frame(data: &[u8]) -> PyResult<(u16, u8, u8, Vec<u8>)> {
    let (transaction_id, frame) = ModbusDecoder::decode_tcp(data)
        .map_err(|e| PyErr::new::<pyo3::exceptions::PyRuntimeError, _>(e.to_string()))?;
    
    Ok((transaction_id, frame.unit_id, frame.function_code as u8, frame.data.to_vec()))
}

/// Create a read holding registers request
#[pyfunction]
fn create_read_holding_registers_request(unit_id: u8, address: u16, quantity: u16) -> PyResult<Vec<u8>> {
    let request = ModbusRequest::ReadHoldingRegisters { address, quantity };
    let frame = request.to_frame(unit_id);
    let encoded = ModbusEncoder::encode_rtu(&frame)
        .map_err(|e| PyErr::new::<pyo3::exceptions::PyRuntimeError, _>(e.to_string()))?;
    
    Ok(encoded.to_vec())
}

/// Parse read holding registers response
#[pyfunction]
fn parse_read_holding_registers_response(data: &[u8]) -> PyResult<Vec<u16>> {
    let frame = ModbusDecoder::decode_rtu(data)
        .map_err(|e| PyErr::new::<pyo3::exceptions::PyRuntimeError, _>(e.to_string()))?;
    
    let response = ModbusDecoder::decode_response(&frame, FunctionCode::ReadHoldingRegisters)
        .map_err(|e| PyErr::new::<pyo3::exceptions::PyRuntimeError, _>(e.to_string()))?;
    
    match response {
        ModbusResponse::ReadHoldingRegisters(values) => Ok(values),
        ModbusResponse::Exception { function, exception_code } => {
            Err(PyErr::new::<pyo3::exceptions::PyRuntimeError, _>(
                format!("Modbus exception: function={}, code={}", function, exception_code)
            ))
        }
        _ => Err(PyErr::new::<pyo3::exceptions::PyRuntimeError, _>(
            "Unexpected response type"
        ))
    }
}

/// A Python module implemented in Rust for high-performance Modbus operations
#[pymodule]
fn modbus_native(m: &Bound<'_, PyModule>) -> PyResult<()> {
    m.add_function(wrap_pyfunction!(encode_rtu_frame, m)?)?;
    m.add_function(wrap_pyfunction!(decode_rtu_frame, m)?)?;
    m.add_function(wrap_pyfunction!(encode_tcp_frame, m)?)?;
    m.add_function(wrap_pyfunction!(decode_tcp_frame, m)?)?;
    m.add_function(wrap_pyfunction!(create_read_holding_registers_request, m)?)?;
    m.add_function(wrap_pyfunction!(parse_read_holding_registers_response, m)?)?;
    Ok(())
}
