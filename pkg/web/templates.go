package web

const indexTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Metal Enrollment - Dashboard</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: #f5f5f5;
            color: #333;
        }
        .header {
            background: #2c3e50;
            color: white;
            padding: 1.5rem 2rem;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .header h1 { font-size: 1.5rem; }
        .container {
            max-width: 1400px;
            margin: 2rem auto;
            padding: 0 2rem;
        }
        .stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1rem;
            margin-bottom: 2rem;
        }
        .stat-card {
            background: white;
            padding: 1.5rem;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .stat-card h3 {
            font-size: 0.875rem;
            color: #666;
            margin-bottom: 0.5rem;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        .stat-card .value {
            font-size: 2rem;
            font-weight: bold;
            color: #2c3e50;
        }
        .machines-table {
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        .table-header {
            padding: 1.5rem;
            border-bottom: 1px solid #e0e0e0;
        }
        .table-header h2 {
            font-size: 1.25rem;
        }
        table {
            width: 100%;
            border-collapse: collapse;
        }
        th, td {
            padding: 1rem 1.5rem;
            text-align: left;
        }
        th {
            background: #f8f9fa;
            font-weight: 600;
            font-size: 0.875rem;
            color: #666;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        tr:not(:last-child) td {
            border-bottom: 1px solid #f0f0f0;
        }
        tbody tr:hover {
            background: #f8f9fa;
        }
        .status-badge {
            display: inline-block;
            padding: 0.25rem 0.75rem;
            border-radius: 12px;
            font-size: 0.75rem;
            font-weight: 600;
            text-transform: uppercase;
        }
        .status-enrolled { background: #e3f2fd; color: #1976d2; }
        .status-configured { background: #fff3e0; color: #f57c00; }
        .status-building { background: #fce4ec; color: #c2185b; }
        .status-ready { background: #e8f5e9; color: #388e3c; }
        .status-provisioned { background: #f3e5f5; color: #7b1fa2; }
        .status-failed { background: #ffebee; color: #d32f2f; }
        .btn {
            padding: 0.5rem 1rem;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 0.875rem;
            text-decoration: none;
            display: inline-block;
        }
        .btn-primary {
            background: #2c3e50;
            color: white;
        }
        .btn-primary:hover {
            background: #34495e;
        }
        .btn-secondary {
            background: #ecf0f1;
            color: #2c3e50;
        }
        .btn-secondary:hover {
            background: #bdc3c7;
        }
        .actions {
            display: flex;
            gap: 0.5rem;
        }
        .empty-state {
            padding: 4rem 2rem;
            text-align: center;
            color: #999;
        }
        .hardware-summary {
            font-size: 0.875rem;
            color: #666;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>⚙️ Metal Enrollment Dashboard</h1>
    </div>

    <div class="container">
        <div class="stats">
            <div class="stat-card">
                <h3>Total Machines</h3>
                <div class="value">{{.TotalMachines}}</div>
            </div>
            <div class="stat-card">
                <h3>Enrolled</h3>
                <div class="value">{{.EnrolledCount}}</div>
            </div>
            <div class="stat-card">
                <h3>Ready</h3>
                <div class="value">{{.ReadyCount}}</div>
            </div>
            <div class="stat-card">
                <h3>Building</h3>
                <div class="value">{{.BuildingCount}}</div>
            </div>
        </div>

        <div class="machines-table">
            <div class="table-header">
                <h2>Enrolled Machines</h2>
            </div>
            {{if .Machines}}
            <table>
                <thead>
                    <tr>
                        <th>Service Tag</th>
                        <th>Hostname</th>
                        <th>Hardware</th>
                        <th>Status</th>
                        <th>Enrolled</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Machines}}
                    <tr>
                        <td><strong>{{.ServiceTag}}</strong></td>
                        <td>{{if .Hostname}}{{.Hostname}}{{else}}<em>Not set</em>{{end}}</td>
                        <td class="hardware-summary">
                            {{.Hardware.CPU.Model}}<br>
                            <small>{{.Hardware.Memory.TotalGB}} GB RAM • {{len .Hardware.Disks}} disk(s)</small>
                        </td>
                        <td><span class="status-badge status-{{.Status}}">{{.Status}}</span></td>
                        <td>{{.EnrolledAt.Format "2006-01-02"}}</td>
                        <td>
                            <div class="actions">
                                <a href="/machines/{{.ID}}" class="btn btn-secondary">View</a>
                                {{if .NixOSConfig}}
                                <a href="/machines/{{.ID}}/build" class="btn btn-primary">Build</a>
                                {{end}}
                            </div>
                        </td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
            {{else}}
            <div class="empty-state">
                <p>No machines enrolled yet. Boot a machine with PXE to get started.</p>
            </div>
            {{end}}
        </div>
    </div>
</body>
</html>`

const machineTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Machine.ServiceTag}} - Metal Enrollment</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: #f5f5f5;
            color: #333;
        }
        .header {
            background: #2c3e50;
            color: white;
            padding: 1.5rem 2rem;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .header h1 { font-size: 1.5rem; }
        .breadcrumb {
            margin-top: 0.5rem;
            font-size: 0.875rem;
        }
        .breadcrumb a { color: #3498db; text-decoration: none; }
        .container {
            max-width: 1200px;
            margin: 2rem auto;
            padding: 0 2rem;
        }
        .card {
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 1.5rem;
            overflow: hidden;
        }
        .card-header {
            padding: 1.5rem;
            border-bottom: 1px solid #e0e0e0;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .card-header h2 { font-size: 1.25rem; }
        .card-body {
            padding: 1.5rem;
        }
        .form-group {
            margin-bottom: 1.5rem;
        }
        .form-group label {
            display: block;
            margin-bottom: 0.5rem;
            font-weight: 600;
            font-size: 0.875rem;
            color: #555;
        }
        .form-group input,
        .form-group textarea {
            width: 100%;
            padding: 0.75rem;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-family: inherit;
            font-size: 0.875rem;
        }
        .form-group textarea {
            font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
            min-height: 300px;
        }
        .btn {
            padding: 0.75rem 1.5rem;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 0.875rem;
            font-weight: 600;
        }
        .btn-primary {
            background: #2c3e50;
            color: white;
        }
        .btn-primary:hover {
            background: #34495e;
        }
        .info-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 1.5rem;
        }
        .info-item {
            padding: 1rem;
            background: #f8f9fa;
            border-radius: 4px;
        }
        .info-item label {
            display: block;
            font-size: 0.75rem;
            color: #666;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            margin-bottom: 0.5rem;
        }
        .info-item .value {
            font-size: 1rem;
            font-weight: 600;
            color: #2c3e50;
        }
        .hardware-list {
            list-style: none;
        }
        .hardware-list li {
            padding: 0.75rem 0;
            border-bottom: 1px solid #f0f0f0;
        }
        .hardware-list li:last-child {
            border-bottom: none;
        }
        .hardware-list strong {
            display: block;
            margin-bottom: 0.25rem;
        }
        .hardware-list small {
            color: #666;
        }
        .status-badge {
            display: inline-block;
            padding: 0.25rem 0.75rem;
            border-radius: 12px;
            font-size: 0.75rem;
            font-weight: 600;
            text-transform: uppercase;
        }
        .status-enrolled { background: #e3f2fd; color: #1976d2; }
        .status-configured { background: #fff3e0; color: #f57c00; }
        .status-building { background: #fce4ec; color: #c2185b; }
        .status-ready { background: #e8f5e9; color: #388e3c; }
    </style>
</head>
<body>
    <div class="header">
        <h1>{{.Machine.ServiceTag}}</h1>
        <div class="breadcrumb">
            <a href="/">← Back to Dashboard</a>
        </div>
    </div>

    <div class="container">
        <div class="card">
            <div class="card-header">
                <h2>Machine Information</h2>
                <span class="status-badge status-{{.Machine.Status}}">{{.Machine.Status}}</span>
            </div>
            <div class="card-body">
                <div class="info-grid">
                    <div class="info-item">
                        <label>Service Tag</label>
                        <div class="value">{{.Machine.ServiceTag}}</div>
                    </div>
                    <div class="info-item">
                        <label>MAC Address</label>
                        <div class="value">{{.Machine.MACAddress}}</div>
                    </div>
                    <div class="info-item">
                        <label>Enrolled At</label>
                        <div class="value">{{.Machine.EnrolledAt.Format "2006-01-02 15:04"}}</div>
                    </div>
                    {{if .Machine.LastSeenAt}}
                    <div class="info-item">
                        <label>Last Seen</label>
                        <div class="value">{{.Machine.LastSeenAt.Format "2006-01-02 15:04"}}</div>
                    </div>
                    {{end}}
                </div>
            </div>
        </div>

        <div class="card">
            <div class="card-header">
                <h2>Hardware Details</h2>
            </div>
            <div class="card-body">
                <div class="info-grid">
                    <div class="info-item">
                        <label>Manufacturer</label>
                        <div class="value">{{.Machine.Hardware.Manufacturer}}</div>
                    </div>
                    <div class="info-item">
                        <label>Model</label>
                        <div class="value">{{.Machine.Hardware.Model}}</div>
                    </div>
                </div>

                <h3 style="margin: 2rem 0 1rem;">CPU</h3>
                <div class="info-grid">
                    <div class="info-item">
                        <label>Model</label>
                        <div class="value">{{.Machine.Hardware.CPU.Model}}</div>
                    </div>
                    <div class="info-item">
                        <label>Configuration</label>
                        <div class="value">{{.Machine.Hardware.CPU.Sockets}} socket(s) × {{.Machine.Hardware.CPU.Cores}} cores × {{.Machine.Hardware.CPU.Threads}} threads</div>
                    </div>
                </div>

                <h3 style="margin: 2rem 0 1rem;">Memory</h3>
                <div class="info-item">
                    <label>Total</label>
                    <div class="value">{{printf "%.2f" .Machine.Hardware.Memory.TotalGB}} GB</div>
                </div>

                <h3 style="margin: 2rem 0 1rem;">Disks</h3>
                <ul class="hardware-list">
                    {{range .Machine.Hardware.Disks}}
                    <li>
                        <strong>{{.Device}}</strong>
                        <small>{{.Model}} • {{printf "%.2f" .SizeGB}} GB • {{.Type}}</small>
                    </li>
                    {{end}}
                </ul>

                <h3 style="margin: 2rem 0 1rem;">Network Interfaces</h3>
                <ul class="hardware-list">
                    {{range .Machine.Hardware.NICs}}
                    <li>
                        <strong>{{.Name}}</strong>
                        <small>{{.MACAddress}} • {{.Speed}} • {{.Driver}}</small>
                    </li>
                    {{end}}
                </ul>
            </div>
        </div>

        <div class="card">
            <div class="card-header">
                <h2>Configuration</h2>
            </div>
            <div class="card-body">
                <form method="POST" action="/machines/{{.Machine.ID}}/update">
                    <div class="form-group">
                        <label for="hostname">Hostname</label>
                        <input type="text" id="hostname" name="hostname" value="{{.Machine.Hostname}}" placeholder="server01">
                    </div>

                    <div class="form-group">
                        <label for="description">Description</label>
                        <input type="text" id="description" name="description" value="{{.Machine.Description}}" placeholder="Production web server">
                    </div>

                    <div class="form-group">
                        <label for="nixos_config">NixOS Configuration</label>
                        <textarea id="nixos_config" name="nixos_config" placeholder="# Enter NixOS configuration here...">{{.Machine.NixOSConfig}}</textarea>
                    </div>

                    <button type="submit" class="btn btn-primary">Save Configuration</button>
                </form>
            </div>
        </div>
    </div>
</body>
</html>`
