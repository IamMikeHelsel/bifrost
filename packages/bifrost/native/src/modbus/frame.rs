'''
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Header {
    pub transaction_id: u16,
    pub protocol_id: u16,
    pub length: u16,
    pub unit_id: u8,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum FunctionCode {
    ReadHoldingRegisters = 0x03,
    WriteSingleRegister = 0x06,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Request {
    pub header: Header,
    pub function: FunctionCode,
    pub address: u16,
    pub count: u16,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Response {
    pub header: Header,
    pub function: FunctionCode,
    pub data: Vec<u8>,
}
'''