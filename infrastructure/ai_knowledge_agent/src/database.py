import os
import logging
import json
import time
from contextlib import contextmanager

import psycopg2
from psycopg2 import pool, OperationalError
from pgvector.psycopg2 import register_vector

logger = logging.getLogger("ai_knowledge_agent")

class Database:
    def __init__(self):
        self.db_pool = None
        self.embedding_dim = 1024  # mxbai-embed-large dimension

        self.dsn = {
            "host": os.getenv("PGHOST", "postgres"),
            "port": os.getenv("PGPORT", "5432"),
            "dbname": os.getenv("PGDATABASE", "nas_db"),
            "user": os.getenv("PGUSER", "nas_user"),
            "password": self._get_db_password(),
        }

    def _get_db_password(self):
        """Read database password from secret file or env var."""
        pwd_file = os.getenv("PGPASSWORD_FILE") or os.getenv("POSTGRES_PASSWORD_FILE")
        if pwd_file and os.path.exists(pwd_file):
            try:
                with open(pwd_file, 'r') as f:
                    return f.read().strip()
            except Exception as e:
                logger.warning("Failed to read password file: %s", e)
        return os.getenv("PGPASSWORD", "")

    def init_pool(self, max_retries=5, base_delay=1.0):
        """Initialize database connection pool with retry logic."""
        for attempt in range(1, max_retries + 1):
            try:
                self.db_pool = psycopg2.pool.SimpleConnectionPool(
                    minconn=1,
                    maxconn=10,
                    **self.dsn
                )
                logger.info("Database connection pool initialized (attempt %d/%d).", attempt, max_retries)
                
                # Check schema
                with self.get_connection() as conn:
                     with conn.cursor() as cur:
                        register_vector(conn)
                        # Schema is already created by SQL migration - just verify it exists
                        cur.execute("""
                            SELECT EXISTS (
                                SELECT FROM information_schema.tables
                                WHERE table_name = 'file_embeddings'
                            )
                        """)
                        exists = cur.fetchone()[0]
                        if not exists:
                            logger.error("file_embeddings table does not exist!")
                            return False
                logger.info("Database schema verified.")
                return True
                
            except OperationalError as e:
                delay = base_delay * (2 ** (attempt - 1))
                logger.warning(
                    "DB connection failed (attempt %d/%d): %s. Retrying in %.1fs...",
                    attempt, max_retries, e, delay
                )
                if attempt < max_retries:
                    time.sleep(delay)
                else:
                    logger.error("Failed to connect to database after %d attempts", max_retries)
                    return False
                    
            except Exception as e:
                logger.error("Unexpected error initializing DB pool: %s", e)
                return False
        
        return False

    @contextmanager
    def get_connection(self):
        if not self.db_pool:
            logger.warning("Database pool not initialized. Attempting lazy initialization...")
            if not self.init_pool(max_retries=3):
                 raise Exception("Database pool not initialized (lazy init failed)")
        
        conn = self.db_pool.getconn()
        try:
            register_vector(conn) # Ensure vector type is registered on this connection
            yield conn
        finally:
            self.db_pool.putconn(conn)

    # === Repository Methods ===

    def get_existing_files(self):
        """Get set of already indexed file_ids."""
        existing_files = set()
        try:
            with self.get_connection() as conn:
                with conn.cursor() as cur:
                    cur.execute("SELECT file_id FROM file_embeddings")
                    existing_files = {row[0] for row in cur.fetchall()}
        except Exception as e:
            logger.warning("Could not check existing files: %s", e)
        return existing_files

    def save_embedding(self, file_id, content, embedding, metadata):
        """Save document embedding to database."""
        with self.get_connection() as conn:
            with conn.cursor() as cur:
                cur.execute("""
                    INSERT INTO file_embeddings (file_id, chunk_index, content, embedding, metadata)
                    VALUES (%s, %s, %s, %s, %s)
                    ON CONFLICT (file_id, chunk_index) DO UPDATE SET
                        content = EXCLUDED.content,
                        embedding = EXCLUDED.embedding,
                        metadata = EXCLUDED.metadata,
                        created_at = CURRENT_TIMESTAMP
                """, (file_id, 0, content, embedding, json.dumps(metadata)))
                conn.commit()

    def delete_embeddings(self, file_id=None, file_path=None):
        """Delete embeddings by file_id or file_path."""
        if not file_id and not file_path:
            return 0
            
        with self.get_connection() as conn:
            with conn.cursor() as cur:
                if file_id:
                    cur.execute("DELETE FROM file_embeddings WHERE file_id = %s", (file_id,))
                else:
                    cur.execute("DELETE FROM file_embeddings WHERE file_path = %s", (file_path,))
                
                deleted_count = cur.rowcount
                conn.commit()
        return deleted_count

    def search_similar(self, query_embedding, limit=10):
        """Search for similar documents using vector similarity."""
        with self.get_connection() as conn:
            with conn.cursor() as cur:
                cur.execute("""
                    SELECT file_id, content, metadata,
                           1 - (embedding <=> %s::vector) as similarity
                    FROM file_embeddings
                    WHERE (metadata->>'file_path') LIKE '/mnt/data/%%'
                      AND (metadata->>'file_path') NOT LIKE '%%/.trash/%%'
                    ORDER BY embedding <=> %s::vector
                    LIMIT %s
                """, (query_embedding, query_embedding, limit))

                results = []
                for row in cur.fetchall():
                    metadata = row[2] or {}
                    results.append({
                        "file_id": row[0],
                        "file_path": metadata.get("file_path", "unknown"),
                        "content": row[1],
                        "similarity": float(row[3])
                    })
        return results

    def get_index_stats(self):
        """Get simple statistics about the index."""
        indexed_files = 0
        try:
            with self.get_connection() as conn:
                with conn.cursor() as cur:
                    cur.execute("SELECT COUNT(*) FROM file_embeddings")
                    indexed_files = cur.fetchone()[0]
        except Exception:
            pass
        return indexed_files

    def list_all_vectors(self):
        """List all file IDs in the vector database."""
        with self.get_connection() as conn:
            with conn.cursor() as cur:
                cur.execute("SELECT DISTINCT file_id FROM file_embeddings")
                return [row[0] for row in cur.fetchall()]
