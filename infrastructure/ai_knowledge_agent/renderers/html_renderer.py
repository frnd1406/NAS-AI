"""
HTML renderer using Jinja2 templates.
"""

from pathlib import Path
from typing import Tuple, Any

from jinja2 import Environment, FileSystemLoader

from renderers.base import Renderer
from data_models import InvoiceData, LogData, EmailData


class HtmlRenderer(Renderer):
    """Renders documents to HTML using Jinja2 templates."""

    def __init__(self, template_dir: str = None):
        if template_dir is None:
            template_dir = Path(__file__).parent.parent / "templates"
        
        self.env = Environment(
            loader=FileSystemLoader(str(template_dir)),
            trim_blocks=True,
            lstrip_blocks=True,
            autoescape=True,
        )

    @property
    def format_name(self) -> str:
        return "html"

    def render(self, data: Any) -> Tuple[bytes, str]:
        """Render data to HTML."""
        if isinstance(data, InvoiceData):
            template = self.env.get_template("invoice_table.html.j2")
            content = template.render(invoice=data)
        elif isinstance(data, LogData):
            template = self.env.get_template("log.html.j2")
            content = template.render(log=data)
        elif isinstance(data, EmailData):
            template = self.env.get_template("email.html.j2")
            content = template.render(email=data)
        else:
            raise ValueError(f"Unknown data type: {type(data)}")

        return content.encode('utf-8'), ".html"
