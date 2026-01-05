<script>
  import { onMount } from 'svelte';
  import { IsConfigured, SaveConfig, NeedsAuth, StartAuth, WaitForAuthCode, GetProjects, ImportTasks, Logout, CreateProject } from '../wailsjs/go/main/App.js';

  let screen = 'loading'; // loading, config, auth, main
  let clientId = '';
  let clientSecret = '';
  let projects = [];
  let selectedProjectId = '';
  let isDragOver = false;
  let importStatus = null;
  let isImporting = false;
  let isAuthenticating = false;
  let configError = '';
  let authError = '';
  let showCreateList = false;
  let newListName = '';
  let isCreatingList = false;
  let isRefreshing = false;
  let createListError = '';

  const jsonExample = `[
  {
    "title": "Buy groceries",
    "content": "Weekly shopping",
    "subtasks": [
      {"title": "Milk"},
      {"title": "Eggs"},
      {"title": "Bread"}
    ]
  },
  {
    "title": "Project tasks",
    "subtasks": [
      {
        "title": "Research",
        "startDate": "2024-01-10",
        "dueDate": "2024-01-15"
      },
      {"title": "Write report"}
    ]
  }
]`;

  onMount(async () => {
    await checkStatus();
  });

  async function checkStatus() {
    try {
      const configured = await IsConfigured();
      if (!configured) {
        screen = 'config';
        return;
      }

      const needsAuth = await NeedsAuth();
      if (needsAuth) {
        screen = 'auth';
        return;
      }

      await loadProjects();
      screen = 'main';
    } catch (err) {
      console.error('Error checking status:', err);
      screen = 'config';
    }
  }

  async function saveConfiguration() {
    if (!clientId.trim() || !clientSecret.trim()) {
      configError = 'Please enter both Client ID and Client Secret';
      return;
    }

    try {
      configError = '';
      await SaveConfig(clientId.trim(), clientSecret.trim());
      screen = 'auth';
    } catch (err) {
      configError = `Failed to save configuration: ${err}`;
    }
  }

  async function startAuthentication() {
    try {
      isAuthenticating = true;
      authError = '';
      await StartAuth();
      await WaitForAuthCode();
      await loadProjects();
      screen = 'main';
    } catch (err) {
      authError = `Authentication failed: ${err}`;
    } finally {
      isAuthenticating = false;
    }
  }

  async function loadProjects() {
    try {
      projects = await GetProjects();
      if (projects.length > 0 && !selectedProjectId) {
        selectedProjectId = projects[0].id;
      }
    } catch (err) {
      console.error('Failed to load projects:', err);
      throw err;
    }
  }

  async function refreshProjects() {
    try {
      isRefreshing = true;
      await loadProjects();
    } catch (err) {
      console.error('Failed to refresh projects:', err);
    } finally {
      isRefreshing = false;
    }
  }

  async function handleCreateList() {
    if (!newListName.trim()) {
      createListError = 'Please enter a list name';
      return;
    }

    try {
      isCreatingList = true;
      createListError = '';
      const newProject = await CreateProject(newListName.trim());
      await loadProjects();
      selectedProjectId = newProject.id;
      newListName = '';
      showCreateList = false;
    } catch (err) {
      createListError = `Failed to create list: ${err}`;
    } finally {
      isCreatingList = false;
    }
  }

  async function handleLogout() {
    try {
      await Logout();
      screen = 'auth';
    } catch (err) {
      console.error('Logout failed:', err);
    }
  }

  function handleDragOver(event) {
    event.preventDefault();
    isDragOver = true;
  }

  function handleDragLeave(event) {
    event.preventDefault();
    isDragOver = false;
  }

  async function handleDrop(event) {
    event.preventDefault();
    isDragOver = false;

    const files = event.dataTransfer.files;
    if (files.length === 0) return;

    const file = files[0];
    await processFile(file);
  }

  async function handleFileSelect(event) {
    const files = event.target.files;
    if (files.length === 0) return;

    const file = files[0];
    await processFile(file);
  }

  async function processFile(file) {
    const fileName = file.name.toLowerCase();
    let fileType = '';

    if (fileName.endsWith('.csv')) {
      fileType = 'csv';
    } else if (fileName.endsWith('.json')) {
      fileType = 'json';
    } else {
      importStatus = {
        success: false,
        message: 'Unsupported file type. Please use CSV or JSON files.'
      };
      return;
    }

    if (!selectedProjectId) {
      importStatus = {
        success: false,
        message: 'Please select a project first.'
      };
      return;
    }

    try {
      isImporting = true;
      importStatus = null;

      const content = await file.text();
      const result = await ImportTasks(selectedProjectId, content, fileType);
      importStatus = result;
    } catch (err) {
      importStatus = {
        success: false,
        message: `Import failed: ${err}`
      };
    } finally {
      isImporting = false;
    }
  }
</script>

<main>
  {#if screen === 'loading'}
    <div class="container">
      <h1>TickTickUp</h1>
      <p>Loading...</p>
    </div>

  {:else if screen === 'config'}
    <div class="container">
      <h1>TickTickUp</h1>
      <p class="subtitle">Configure your TickTick API credentials</p>

      <div class="config-form">
        <p class="help-text">
          Get your API credentials from the
          <a href="https://developer.ticktick.com/manage" target="_blank" rel="noopener">
            TickTick Developer Portal
          </a>
        </p>

        <div class="form-group">
          <label for="clientId">Client ID</label>
          <input
            type="text"
            id="clientId"
            bind:value={clientId}
            placeholder="Enter your Client ID"
          />
        </div>

        <div class="form-group">
          <label for="clientSecret">Client Secret</label>
          <input
            type="password"
            id="clientSecret"
            bind:value={clientSecret}
            placeholder="Enter your Client Secret"
          />
        </div>

        {#if configError}
          <p class="error">{configError}</p>
        {/if}

        <button class="btn primary" on:click={saveConfiguration}>
          Save Configuration
        </button>
      </div>
    </div>

  {:else if screen === 'auth'}
    <div class="container">
      <h1>TickTickUp</h1>
      <p class="subtitle">Authentication Required</p>

      <div class="auth-section">
        <p>Please authenticate with TickTick to continue.</p>

        {#if authError}
          <p class="error">{authError}</p>
        {/if}

        <button
          class="btn primary"
          on:click={startAuthentication}
          disabled={isAuthenticating}
        >
          {isAuthenticating ? 'Authenticating...' : 'Authenticate with TickTick'}
        </button>
      </div>
    </div>

  {:else if screen === 'main'}
    <div class="container">
      <div class="header">
        <h1>TickTickUp</h1>
        <button class="btn-link" on:click={handleLogout}>Logout</button>
      </div>

      <div class="project-selector">
        <div class="project-row">
          <label for="project">Import to:</label>
          <select id="project" bind:value={selectedProjectId}>
            {#each projects as project}
              <option value={project.id}>{project.name}</option>
            {/each}
          </select>
          <button
            class="btn-icon"
            on:click={refreshProjects}
            disabled={isRefreshing}
            title="Refresh lists"
          >
            {isRefreshing ? '...' : '↻'}
          </button>
          <button
            class="btn-icon"
            on:click={() => { showCreateList = !showCreateList; createListError = ''; }}
            title="Create new list"
          >
            +
          </button>
        </div>

        {#if showCreateList}
          <div class="create-list-form">
            <input
              type="text"
              bind:value={newListName}
              placeholder="New list name"
              on:keydown={(e) => e.key === 'Enter' && handleCreateList()}
            />
            <button
              class="btn small primary"
              on:click={handleCreateList}
              disabled={isCreatingList}
            >
              {isCreatingList ? 'Creating...' : 'Create'}
            </button>
            <button
              class="btn small secondary"
              on:click={() => { showCreateList = false; newListName = ''; createListError = ''; }}
            >
              Cancel
            </button>
            {#if createListError}
              <p class="error small">{createListError}</p>
            {/if}
          </div>
        {/if}
      </div>

      <div
        class="drop-zone"
        class:drag-over={isDragOver}
        on:dragover={handleDragOver}
        on:dragleave={handleDragLeave}
        on:drop={handleDrop}
        role="button"
        tabindex="0"
      >
        <div class="drop-content">
          {#if isImporting}
            <p class="drop-text">Importing tasks...</p>
          {:else}
            <p class="drop-text">Drag & drop CSV or JSON file here</p>
            <p class="drop-subtext">or</p>
            <label class="btn secondary file-btn">
              Browse Files
              <input
                type="file"
                accept=".csv,.json"
                on:change={handleFileSelect}
                style="display: none;"
              />
            </label>
          {/if}
        </div>
      </div>

      {#if importStatus}
        <div class="status" class:success={importStatus.success} class:error={!importStatus.success}>
          <p class="status-message">{importStatus.message}</p>
          {#if importStatus.errors && importStatus.errors.length > 0}
            <details>
              <summary>Show errors ({importStatus.errors.length})</summary>
              <ul class="error-list">
                {#each importStatus.errors as error}
                  <li>{error}</li>
                {/each}
              </ul>
            </details>
          {/if}
        </div>
      {/if}

      <div class="format-help">
        <details>
          <summary>File format help</summary>
          <div class="format-info">
            <h4>CSV Format</h4>
            <p>Columns: title (required), content, dueDate, tags, subtasks</p>
            <p class="format-note">Subtasks format: title|startDate|dueDate (dates optional), separated by semicolons</p>
            <pre>title,content,dueDate,tags,subtasks
Buy groceries,Weekly shopping,2024-01-15,"shopping","Milk;Eggs;Bread"
Project work,Important,2024-01-20,work,"Research|2024-01-10|2024-01-15;Write report||2024-01-18;Review"</pre>

            <h4>JSON Format</h4>
            <p>Array of tasks with optional subtasks (supports startDate/dueDate):</p>
            <pre>{jsonExample}</pre>
          </div>
        </details>
      </div>
    </div>
  {/if}
</main>

<style>
  main {
    display: flex;
    justify-content: center;
    align-items: flex-start;
    min-height: 100vh;
    padding: 40px 20px;
  }

  .container {
    max-width: 600px;
    width: 100%;
  }

  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 30px;
  }

  h1 {
    margin: 0;
    font-size: 2rem;
    color: #fff;
  }

  .subtitle {
    color: #aaa;
    margin: 10px 0 30px;
  }

  .help-text {
    color: #888;
    margin-bottom: 20px;
  }

  .help-text a {
    color: #4a9eff;
    text-decoration: none;
  }

  .help-text a:hover {
    text-decoration: underline;
  }

  .config-form, .auth-section {
    background: rgba(255, 255, 255, 0.05);
    border-radius: 12px;
    padding: 30px;
  }

  .form-group {
    margin-bottom: 20px;
    text-align: left;
  }

  .form-group label {
    display: block;
    margin-bottom: 8px;
    color: #ccc;
  }

  input[type="text"],
  input[type="password"] {
    width: 100%;
    padding: 12px;
    border: 1px solid #444;
    border-radius: 6px;
    background: rgba(255, 255, 255, 0.1);
    color: white;
    font-size: 1rem;
    box-sizing: border-box;
  }

  input[type="text"]:focus,
  input[type="password"]:focus {
    outline: none;
    border-color: #4a9eff;
  }

  .btn {
    padding: 12px 24px;
    border: none;
    border-radius: 6px;
    font-size: 1rem;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn.primary {
    background: #4a9eff;
    color: white;
  }

  .btn.primary:hover:not(:disabled) {
    background: #3a8eef;
  }

  .btn.primary:disabled {
    background: #666;
    cursor: not-allowed;
  }

  .btn.secondary {
    background: rgba(255, 255, 255, 0.1);
    color: white;
    border: 1px solid #444;
  }

  .btn.secondary:hover {
    background: rgba(255, 255, 255, 0.15);
  }

  .btn-link {
    background: none;
    border: none;
    color: #888;
    cursor: pointer;
    font-size: 0.9rem;
  }

  .btn-link:hover {
    color: #ccc;
  }

  .project-selector {
    margin-bottom: 30px;
    text-align: left;
  }

  .project-row {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-wrap: wrap;
  }

  .project-selector label {
    color: #ccc;
  }

  select {
    padding: 10px 15px;
    border: 1px solid #444;
    border-radius: 6px;
    background: rgba(255, 255, 255, 0.1);
    color: white;
    font-size: 1rem;
    min-width: 200px;
  }

  select:focus {
    outline: none;
    border-color: #4a9eff;
  }

  .btn-icon {
    width: 36px;
    height: 36px;
    padding: 0;
    border: 1px solid #444;
    border-radius: 6px;
    background: rgba(255, 255, 255, 0.1);
    color: white;
    font-size: 1.2rem;
    cursor: pointer;
    transition: all 0.2s;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .btn-icon:hover:not(:disabled) {
    background: rgba(255, 255, 255, 0.15);
    border-color: #4a9eff;
  }

  .btn-icon:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .create-list-form {
    margin-top: 15px;
    display: flex;
    align-items: center;
    gap: 10px;
    flex-wrap: wrap;
  }

  .create-list-form input {
    padding: 8px 12px;
    border: 1px solid #444;
    border-radius: 6px;
    background: rgba(255, 255, 255, 0.1);
    color: white;
    font-size: 0.9rem;
    min-width: 180px;
  }

  .create-list-form input:focus {
    outline: none;
    border-color: #4a9eff;
  }

  .btn.small {
    padding: 8px 16px;
    font-size: 0.9rem;
  }

  .error.small {
    font-size: 0.85rem;
    margin: 5px 0 0;
    width: 100%;
  }

  .drop-zone {
    border: 2px dashed #444;
    border-radius: 12px;
    padding: 60px 30px;
    text-align: center;
    transition: all 0.2s;
    cursor: pointer;
  }

  .drop-zone:hover,
  .drop-zone.drag-over {
    border-color: #4a9eff;
    background: rgba(74, 158, 255, 0.05);
  }

  .drop-content {
    pointer-events: none;
  }

  .drop-text {
    font-size: 1.2rem;
    color: #ccc;
    margin: 0 0 10px;
  }

  .drop-subtext {
    color: #666;
    margin: 10px 0;
  }

  .file-btn {
    pointer-events: auto;
    display: inline-block;
  }

  .status {
    margin-top: 30px;
    padding: 20px;
    border-radius: 8px;
    text-align: left;
  }

  .status.success {
    background: rgba(46, 204, 113, 0.15);
    border: 1px solid rgba(46, 204, 113, 0.3);
  }

  .status.error {
    background: rgba(231, 76, 60, 0.15);
    border: 1px solid rgba(231, 76, 60, 0.3);
  }

  .status-message {
    margin: 0;
    font-weight: 500;
  }

  .status.success .status-message {
    color: #2ecc71;
  }

  .status.error .status-message {
    color: #e74c3c;
  }

  .error {
    color: #e74c3c;
    margin: 15px 0;
  }

  .error-list {
    margin: 10px 0;
    padding-left: 20px;
    color: #e74c3c;
    font-size: 0.9rem;
    text-align: left;
  }

  details {
    margin-top: 10px;
  }

  summary {
    cursor: pointer;
    color: #888;
  }

  summary:hover {
    color: #ccc;
  }

  .format-help {
    margin-top: 30px;
    text-align: left;
  }

  .format-info {
    background: rgba(255, 255, 255, 0.05);
    padding: 20px;
    border-radius: 8px;
    margin-top: 10px;
  }

  .format-info h4 {
    margin: 20px 0 10px;
    color: #ccc;
  }

  .format-info h4:first-child {
    margin-top: 0;
  }

  .format-info p {
    color: #888;
    margin: 0 0 10px;
    font-size: 0.9rem;
  }

  .format-info .format-note {
    color: #6aa;
    font-style: italic;
  }

  .format-info pre {
    background: rgba(0, 0, 0, 0.3);
    padding: 10px;
    border-radius: 4px;
    overflow-x: auto;
    font-size: 0.8rem;
    color: #aaa;
  }
</style>
