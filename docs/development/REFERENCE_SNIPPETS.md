# CRITICAL CODE SNIPPETS - Must Have for Next Session

## 🔥 COMPLETE File Handler (Go)
**Location:** `/infrastructure/api/src/handlers/files.go`

```go
package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// FileItem represents a file or directory
type FileItem struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	IsDirectory bool      `json:"isDirectory"`
	Size        int64     `json:"size"`
	Modified    time.Time `json:"modified"`
}

// ListFilesResponse represents the response for listing files
type ListFilesResponse struct {
	Files []FileItem `json:"files"`
	Path  string     `json:"path"`
}

// Base directory for file operations (configurable via env)
var baseDir = "/srv/data"

func init() {
	if dir := os.Getenv("FILES_BASE_DIR"); dir != "" {
		baseDir = dir
	}
}

// sanitizePath ensures the path is within the base directory
func sanitizePath(requestedPath string) (string, error) {
	cleanPath := filepath.Clean(requestedPath)
	fullPath := filepath.Join(baseDir, cleanPath)

	if !strings.HasPrefix(fullPath, baseDir) {
		return "", fmt.Errorf("access denied: path outside base directory")
	}

	return fullPath, nil
}

// ListFiles lists files in a directory
func ListFiles(c *gin.Context) {
	requestedPath := c.DefaultQuery("path", "/")

	fullPath, err := sanitizePath(requestedPath)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "path not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	if !info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is not a directory"})
		return
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	files := make([]FileItem, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		itemPath := filepath.Join(requestedPath, entry.Name())
		files = append(files, FileItem{
			Name:        entry.Name(),
			Path:        itemPath,
			IsDirectory: entry.IsDir(),
			Size:        info.Size(),
			Modified:    info.ModTime(),
		})
	}

	c.JSON(http.StatusOK, ListFilesResponse{
		Files: files,
		Path:  requestedPath,
	})
}

// CreateDirectory creates a new directory
func CreateDirectory(c *gin.Context) {
	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.Path == "" || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path and name required"})
		return
	}

	dirPath := filepath.Join(req.Path, req.Name)
	fullPath, err := sanitizePath(dirPath)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	if err := os.MkdirAll(fullPath, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "directory created successfully",
		"path":    dirPath,
	})
}
```

---

## 🔥 CORS Middleware (Go)
**Location:** `/infrastructure/api/src/main.go`

```go
// CORS middleware - ADD THIS AFTER r := gin.Default()
r.Use(func(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

	if c.Request.Method == "OPTIONS" {
		c.AbortWithStatus(204)
		return
	}

	c.Next()
})
```

---

## 🔥 Route Registration (Go)
**Location:** `/infrastructure/api/src/main.go` in v1 group

```go
// File management endpoints
v1.GET("/files", handlers.ListFiles)
v1.POST("/files/directory", handlers.CreateDirectory)
```

---

## 🔥 Files Store (TypeScript)
**Location:** `/infrastructure/webui/src/state/files.ts`

```typescript
import { create } from 'zustand';
import apiClient from '../services/api/client';

interface FileItem {
    name: string;
    path: string;
    isDirectory: boolean;
    size: number;
    modified: string;
}

interface FilesStore {
    files: FileItem[];
    currentPath: string;
    isLoading: boolean;
    error: string | null;
    fetchFiles: (path?: string) => Promise<void>;
    createDirectory: (path: string, name: string) => Promise<void>;
}

const useFilesStore = create<FilesStore>((set, get) => ({
    files: [],
    currentPath: '/',
    isLoading: false,
    error: null,

    fetchFiles: async (path?: string) => {
        const targetPath = path || get().currentPath;
        set({ isLoading: true, error: null });

        try {
            const response = await apiClient.get('/api/v1/files', {
                params: { path: targetPath },
            });

            set({
                files: response.data.files || [],
                currentPath: targetPath,
                isLoading: false,
            });
        } catch (error: any) {
            set({
                error: error.response?.data?.message || 'Failed to fetch files',
                isLoading: false,
            });
            throw error;
        }
    },

    createDirectory: async (path: string, name: string) => {
        try {
            await apiClient.post('/api/v1/files/directory', {
                path,
                name,
            });

            await get().fetchFiles();
        } catch (error: any) {
            set({
                error: error.response?.data?.message || 'Failed to create directory',
            });
            throw error;
        }
    },
}));

export default useFilesStore;
```

---

## 🔥 API Client (TypeScript)
**Location:** `/infrastructure/webui/src/services/api/client.ts`

```typescript
import axios, { type AxiosInstance } from 'axios';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export const apiClient: AxiosInstance = axios.create({
    baseURL: API_BASE_URL,
    timeout: 30000,
    headers: {
        'Content-Type': 'application/json',
    },
});

// Request interceptor - Add auth token
apiClient.interceptors.request.use(
    (config) => {
        const token = localStorage.getItem('accessToken');
        if (token && config.headers) {
            config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
    },
    (error) => {
        return Promise.reject(error);
    }
);

export default apiClient;
```

---

## 🔥 Environment Config
**Frontend `.env.local`:**
```bash
VITE_API_URL=http://localhost:8080
```

**Backend Environment:**
```bash
PORT=8080
JWT_SECRET=your-secret-here
FILES_BASE_DIR=/path/to/data
VAULT_TOKEN=stub
```

---

## 🔥 Minimal Files Page (TypeScript)
**Location:** `/infrastructure/webui/src/pages/Files.tsx`

```typescript
import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import useFilesStore from '../state/files';

export default function Files() {
    const { files, currentPath, fetchFiles, createDirectory, isLoading, error } = useFilesStore();
    const [newFolderName, setNewFolderName] = useState('');

    useEffect(() => {
        fetchFiles();
    }, []);

    const handleFileClick = (file: any) => {
        if (file.isDirectory) {
            fetchFiles(file.path);
        }
    };

    const handleCreateFolder = async () => {
        if (!newFolderName) return;
        try {
            await createDirectory(currentPath, newFolderName);
            setNewFolderName('');
        } catch (err) {
            console.error('Create folder failed:', err);
        }
    };

    const handleGoUp = () => {
        const parts = currentPath.split('/').filter(p => p);
        parts.pop();
        const parentPath = '/' + parts.join('/');
        fetchFiles(parentPath);
    };

    return (
        <div>
            <h1>Files</h1>
            <Link to="/dashboard">Back to Dashboard</Link>

            <div>
                <h2>Path: {currentPath}</h2>
                {currentPath !== '/' && (
                    <button onClick={handleGoUp}>Go Up</button>
                )}
            </div>

            <div>
                <input
                    type="text"
                    value={newFolderName}
                    onChange={(e) => setNewFolderName(e.target.value)}
                    placeholder="New folder name"
                />
                <button onClick={handleCreateFolder}>Create Folder</button>
            </div>

            {isLoading && <p>Loading...</p>}
            {error && <p>Error: {error}</p>}

            <ul>
                {files.map((file) => (
                    <li key={file.path} onClick={() => handleFileClick(file)}>
                        {file.isDirectory ? '📁' : '📄'} {file.name}
                    </li>
                ))}
            </ul>
        </div>
    );
}
```

---

**ENDE - Alle kritischen Code-Snippets gesichert!**
