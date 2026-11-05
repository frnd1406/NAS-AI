"""
PDF renderer using ReportLab.
"""

from io import BytesIO
from typing import Tuple, Any

from reportlab.lib import colors
from reportlab.lib.pagesizes import A4
from reportlab.lib.styles import getSampleStyleSheet, ParagraphStyle
from reportlab.lib.units import cm
from reportlab.platypus import SimpleDocTemplate, Paragraph, Spacer, Table, TableStyle

from renderers.base import Renderer
from data_models import InvoiceData, LogData, EmailData


class PdfRenderer(Renderer):
    """Renders documents to PDF using ReportLab."""

    def __init__(self):
        self.styles = getSampleStyleSheet()
        self.styles.add(ParagraphStyle(
            name='GermanBody',
            fontName='Helvetica',
            fontSize=10,
            leading=14,
        ))
        self.styles.add(ParagraphStyle(
            name='InvoiceHeader',
            fontName='Helvetica-Bold',
            fontSize=16,
            spaceAfter=20,
        ))

    @property
    def format_name(self) -> str:
        return "pdf"

    def render(self, data: Any) -> Tuple[bytes, str]:
        """Render data to PDF."""
        buffer = BytesIO()
        doc = SimpleDocTemplate(
            buffer,
            pagesize=A4,
            rightMargin=2*cm,
            leftMargin=2*cm,
            topMargin=2*cm,
            bottomMargin=2*cm,
        )

        if isinstance(data, InvoiceData):
            story = self._render_invoice(data)
        elif isinstance(data, LogData):
            story = self._render_log(data)
        elif isinstance(data, EmailData):
            story = self._render_email(data)
        else:
            raise ValueError(f"Unknown data type: {type(data)}")

        doc.build(story)
        return buffer.getvalue(), ".pdf"

    def _render_invoice(self, invoice: InvoiceData) -> list:
        """Build PDF story for invoice."""
        story = []
        
        # Header
        story.append(Paragraph("RECHNUNG", self.styles['InvoiceHeader']))
        
        # Invoice details
        details = f"""
        <b>Rechnungsnummer:</b> {invoice.invoice_number}<br/>
        <b>Rechnungsdatum:</b> {invoice.invoice_date.strftime('%d.%m.%Y')}<br/>
        <b>Fälligkeitsdatum:</b> {invoice.due_date.strftime('%d.%m.%Y')}
        """
        story.append(Paragraph(details, self.styles['GermanBody']))
        story.append(Spacer(1, 0.5*cm))

        # Company info
        company = f"""
        <b>Von:</b><br/>
        {invoice.company.name}<br/>
        {invoice.company.street}<br/>
        {invoice.company.postcode} {invoice.company.city}<br/>
        Steuernummer: {invoice.company.tax_id}
        """
        story.append(Paragraph(company, self.styles['GermanBody']))
        story.append(Spacer(1, 0.5*cm))

        # Customer info
        customer = f"""
        <b>An:</b><br/>
        {invoice.customer.name}<br/>
        {invoice.customer.street}<br/>
        {invoice.customer.postcode} {invoice.customer.city}
        """
        story.append(Paragraph(customer, self.styles['GermanBody']))
        story.append(Spacer(1, 0.5*cm))

        # Items table
        table_data = [['Pos.', 'Beschreibung', 'Menge', 'Einzelpreis', 'Gesamt']]
        for item in invoice.items:
            table_data.append([
                str(item.position),
                item.name,
                str(item.quantity),
                f"{item.unit_price:.2f}€",
                f"{item.total:.2f}€",
            ])

        table = Table(table_data, colWidths=[1*cm, 8*cm, 2*cm, 3*cm, 3*cm])
        table.setStyle(TableStyle([
            ('BACKGROUND', (0, 0), (-1, 0), colors.grey),
            ('TEXTCOLOR', (0, 0), (-1, 0), colors.whitesmoke),
            ('ALIGN', (2, 0), (-1, -1), 'RIGHT'),
            ('FONTNAME', (0, 0), (-1, 0), 'Helvetica-Bold'),
            ('FONTSIZE', (0, 0), (-1, -1), 9),
            ('BOTTOMPADDING', (0, 0), (-1, 0), 12),
            ('GRID', (0, 0), (-1, -1), 1, colors.black),
        ]))
        story.append(table)
        story.append(Spacer(1, 0.5*cm))

        # Totals
        totals = f"""
        <b>Nettobetrag:</b> {invoice.net_total:.2f}€<br/>
        <b>MwSt. (19%):</b> {invoice.vat:.2f}€<br/>
        <b>Gesamtbetrag:</b> {invoice.gross_total:.2f}€
        """
        story.append(Paragraph(totals, self.styles['GermanBody']))
        story.append(Spacer(1, 0.5*cm))

        # Payment info
        payment = f"""
        <b>Zahlungsinformationen:</b><br/>
        IBAN: {invoice.iban}<br/>
        BIC: {invoice.bic}<br/>
        Verwendungszweck: {invoice.invoice_number}
        """
        story.append(Paragraph(payment, self.styles['GermanBody']))

        return story

    def _render_log(self, log: LogData) -> list:
        """Build PDF story for tech log."""
        story = []
        
        story.append(Paragraph(
            f"Server Log: {log.server_name}",
            self.styles['InvoiceHeader']
        ))
        
        story.append(Paragraph(
            f"Generated: {log.log_date.strftime('%Y-%m-%d')}",
            self.styles['GermanBody']
        ))
        story.append(Spacer(1, 0.5*cm))

        # Log entries
        for entry in log.entries[:20]:  # Limit for PDF
            color = 'red' if entry.level == 'ERROR' else \
                    'orange' if entry.level == 'WARN' else 'black'
            line = f"[{entry.timestamp.strftime('%Y-%m-%d %H:%M:%S')}] " \
                   f"[<font color='{color}'>{entry.level}</font>] " \
                   f"[{entry.server}] {entry.message}"
            story.append(Paragraph(line, self.styles['GermanBody']))

        story.append(Spacer(1, 0.5*cm))
        story.append(Paragraph(
            f"Total Entries: {len(log.entries)} | Uptime: {log.uptime_hours} hours",
            self.styles['GermanBody']
        ))

        return story

    def _render_email(self, email: EmailData) -> list:
        """Build PDF story for email."""
        story = []
        
        story.append(Paragraph(email.subject, self.styles['InvoiceHeader']))
        
        headers = f"""
        <b>Von:</b> {email.sender_name} &lt;{email.sender_email}&gt;<br/>
        <b>An:</b> {email.recipient_name} &lt;{email.recipient_email}&gt;<br/>
        <b>Datum:</b> {email.sent_date.strftime('%d.%m.%Y %H:%M')}
        """
        story.append(Paragraph(headers, self.styles['GermanBody']))
        story.append(Spacer(1, 0.5*cm))

        # Body - convert newlines to <br/>
        body_html = email.body.replace('\n', '<br/>')
        story.append(Paragraph(body_html, self.styles['GermanBody']))

        return story
