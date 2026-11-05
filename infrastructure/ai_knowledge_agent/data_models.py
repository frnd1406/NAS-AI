"""
Data models for the corpus generator pipeline.
Typed dataclasses representing structured document data.
"""

from dataclasses import dataclass, field
from datetime import date, datetime
from typing import List, Optional


@dataclass
class LineItem:
    """Single invoice line item."""
    position: int
    name: str
    quantity: int
    unit_price: float
    total: float


@dataclass
class CompanyData:
    """Company information."""
    name: str
    street: str
    postcode: str
    city: str
    tax_id: str


@dataclass
class CustomerData:
    """Customer information."""
    name: str
    street: str
    postcode: str
    city: str


@dataclass
class InvoiceData:
    """Complete invoice data structure."""
    invoice_number: str
    invoice_date: date
    due_date: date
    company: CompanyData
    customer: CustomerData
    items: List[LineItem]
    net_total: float
    vat: float
    gross_total: float
    iban: str
    bic: str

    def to_dict(self) -> dict:
        """Convert to dictionary for JSON serialization."""
        return {
            "invoice_number": self.invoice_number,
            "invoice_date": self.invoice_date.isoformat(),
            "due_date": self.due_date.isoformat(),
            "company": {
                "name": self.company.name,
                "street": self.company.street,
                "postcode": self.company.postcode,
                "city": self.company.city,
                "tax_id": self.company.tax_id,
            },
            "customer": {
                "name": self.customer.name,
                "street": self.customer.street,
                "postcode": self.customer.postcode,
                "city": self.customer.city,
            },
            "items": [
                {
                    "position": item.position,
                    "name": item.name,
                    "quantity": item.quantity,
                    "unit_price": item.unit_price,
                    "total": item.total,
                }
                for item in self.items
            ],
            "net_total": self.net_total,
            "vat": self.vat,
            "gross_total": self.gross_total,
            "iban": self.iban,
            "bic": self.bic,
        }


@dataclass
class LogEntry:
    """Single log entry."""
    timestamp: datetime
    level: str
    server: str
    message: str


@dataclass
class LogData:
    """Complete tech log data structure."""
    server_name: str
    log_date: datetime
    entries: List[LogEntry]
    uptime_hours: int

    def to_dict(self) -> dict:
        """Convert to dictionary for JSON serialization."""
        return {
            "server_name": self.server_name,
            "log_date": self.log_date.isoformat(),
            "entry_count": len(self.entries),
            "uptime_hours": self.uptime_hours,
            "levels": list(set(e.level for e in self.entries)),
        }


@dataclass
class EmailData:
    """Complete email data structure."""
    sender_name: str
    sender_email: str
    recipient_name: str
    recipient_email: str
    subject: str
    sent_date: datetime
    body: str

    def to_dict(self) -> dict:
        """Convert to dictionary for JSON serialization."""
        return {
            "sender_name": self.sender_name,
            "sender_email": self.sender_email,
            "recipient_name": self.recipient_name,
            "recipient_email": self.recipient_email,
            "subject": self.subject,
            "sent_date": self.sent_date.isoformat(),
        }


@dataclass
class GroundTruthEntry:
    """Single entry in the ground truth registry."""
    filename: str
    doc_type: str  # invoice, log, email
    format: str    # pdf, txt, html
    has_noise: bool
    metadata: dict


@dataclass 
class GroundTruth:
    """Complete ground truth registry."""
    generated_at: datetime
    total_documents: int
    documents: List[GroundTruthEntry] = field(default_factory=list)

    def to_dict(self) -> dict:
        """Convert to dictionary for JSON serialization."""
        return {
            "generated_at": self.generated_at.isoformat(),
            "total_documents": self.total_documents,
            "documents": [
                {
                    "filename": doc.filename,
                    "type": doc.doc_type,
                    "format": doc.format,
                    "has_noise": doc.has_noise,
                    "metadata": doc.metadata,
                }
                for doc in self.documents
            ],
        }
