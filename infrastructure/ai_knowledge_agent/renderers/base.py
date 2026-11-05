"""
Base renderer interface.
"""

from abc import ABC, abstractmethod
from typing import Tuple, Any


class Renderer(ABC):
    """Abstract base class for document renderers."""

    @abstractmethod
    def render(self, data: Any) -> Tuple[bytes, str]:
        """
        Render document data to output format.
        
        Args:
            data: Structured data object (InvoiceData, LogData, EmailData)
            
        Returns:
            Tuple of (content_bytes, file_extension)
        """
        pass

    @property
    @abstractmethod
    def format_name(self) -> str:
        """Return the format name (e.g., 'txt', 'pdf', 'html')."""
        pass
