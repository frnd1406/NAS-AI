import logging
import os
from flask import Flask, jsonify
from werkzeug.exceptions import HTTPException

from src.database import Database
from src.services.rag_service import RAGService
from src.routes.api import api_bp

# Configure Logging
logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
logger = logging.getLogger("ai_knowledge_agent")

def create_app():
    """Application Factory & Composition Root"""
    app = Flask(__name__)
    
    # 1. Initialize Infrastructure
    db = Database()
    
    # 2. Initialize Application Services (Dependency Injection)
    rag_service = RAGService(db)
    
    # 3. Store Service in App Context (for Routes)
    app.config['RAG_SERVICE'] = rag_service
    
    # 4. Register Interface (Routes)
    app.register_blueprint(api_bp)

    # 4b. Global Error Handling
    # SECURITY: never leak exception details (stack traces, SQL, paths,
    # secrets) to clients. Log them server-side, return a generic body.
    @app.errorhandler(HTTPException)
    def handle_http_exception(e):
        # Pass through intentional HTTP statuses (400/403/404/405/...)
        # but normalize the body to JSON.
        return jsonify({"error": e.name}), e.code

    @app.errorhandler(Exception)
    def handle_unexpected_exception(e):
        logger.exception("Unhandled exception while serving request: %s", type(e).__name__)
        return jsonify({"error": "Internal server error"}), 500

    # 5. Startup Logic
    with app.app_context():
        success = db.init_pool()
        if not success:
            logger.error("Failed to initialize database pool on startup")
        
        # Start background tasks
        rag_service.start_background_threads()
        rag_service.prewarm_models()
        
    return app

# Expose app for Gunicorn
app = create_app()

if __name__ == "__main__":
    port = int(os.getenv("PORT", 5000))
    app.run(host="0.0.0.0", port=port, debug=False)
