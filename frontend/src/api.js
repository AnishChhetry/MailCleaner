/**
 * Central API utility for MailCleaner
 * Enhanced with better error handling and modern practices
 */

const API_BASE = process.env.REACT_APP_API_BASE || 'http://localhost:8080';

/**
 * Enhanced request wrapper with better error handling and longer timeout for sync operations
 * @param {string} endpoint - API endpoint path
 * @param {RequestInit} options - Fetch options
 * @returns {Promise<any>} - Parsed JSON response
 */
async function request(endpoint, options = {}) {
  const controller = new AbortController();
  // Longer timeout for sync operations
  const timeout = endpoint.includes('/sync') ? 300000 : 30000; // 5 minutes for sync, 30s for others
  const timeoutId = setTimeout(() => controller.abort(), timeout);

  try {
    const res = await fetch(`${API_BASE}${endpoint}`, {
      ...options,
      signal: controller.signal,
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
    });

    clearTimeout(timeoutId);

    // Handle authentication errors
    if (res.status === 401 && endpoint !== '/healthz') {
      // Attempt logout and redirect
      fetch(`${API_BASE}/logout`, { method: 'POST', credentials: 'include' })
        .finally(() => {
          window.location.href = '/';
        });
      
      throw new Error('Session expired. Please sign in again.');
    }

    // Handle other HTTP errors
    if (!res.ok) {
      let errorMessage = `Request failed with status: ${res.status} ${res.statusText}`;
      
      try {
        const errorBody = await res.json();
        errorMessage = errorBody.error || errorBody.message || errorMessage;
      } catch (parseError) {
        // If we can't parse the error body, use the default message
        console.warn('Could not parse error response:', parseError);
      }
      
      throw new Error(errorMessage);
    }

    // Try to parse response as JSON
    const contentType = res.headers.get('content-type');
    if (contentType && contentType.includes('application/json')) {
      return await res.json();
    } else {
      // Handle non-JSON responses (e.g., empty 204 responses)
      const text = await res.text();
      return text ? { message: text } : { message: 'Success' };
    }

  } catch (error) {
    clearTimeout(timeoutId);
    
    if (error.name === 'AbortError') {
      throw new Error('Request timed out. Please try again.');
    }
    
    if (error instanceof TypeError && error.message.includes('fetch')) {
      throw new Error('Network error. Please check your connection and try again.');
    }
    
    throw error;
  }
}

/**
 * Creates a GET request
 * @param {string} endpoint - API endpoint
 * @param {Record<string, string>} params - URL parameters
 * @returns {Promise<any>}
 */
async function get(endpoint, params = {}) {
  const url = new URL(`${API_BASE}${endpoint}`);
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') {
      url.searchParams.append(key, String(value));
    }
  });
  
  return request(url.pathname + url.search);
}

/**
 * Creates a POST request
 * @param {string} endpoint - API endpoint
 * @param {any} data - Request body data
 * @returns {Promise<any>}
 */
async function post(endpoint, data = null) {
  return request(endpoint, {
    method: 'POST',
    body: data ? JSON.stringify(data) : undefined,
  });
}

/**
 * Creates a PUT request
 * @param {string} endpoint - API endpoint  
 * @param {any} data - Request body data
 * @returns {Promise<any>}
 */
async function put(endpoint, data) {
  return request(endpoint, {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}

/**
 * Creates a DELETE request
 * @param {string} endpoint - API endpoint
 * @returns {Promise<any>}
 */
async function del(endpoint) {
  return request(endpoint, { method: 'DELETE' });
}

// =============================================================================
// AUTH ENDPOINTS
// =============================================================================

export const loginUrl = `${API_BASE}/auth/google/login`;

/**
 * Logout the current user
 * @returns {Promise<any>}
 */
export const logout = () => post('/logout');

/**
 * Check if user is authenticated
 * @returns {Promise<any>}
 */
export const healthz = () => get('/healthz');

// =============================================================================
// EMAIL ENDPOINTS
// =============================================================================

export const fetchEmails = () => get('/emails');
export const fetchEmailsPaginated = (page = 1, pageSize = 10, filter = '') => 
  get('/emails/paginated', { page: String(page), pageSize: String(pageSize), filter });
export const syncEmails = () => post('/emails/sync');
export const syncHistory = () => post('/emails/sync-history');
export const getSyncProgress = () => get('/emails/sync/progress');
export const fetchEmailDetails = (id) => get(`/emails/${id}`);
export const deleteEmail = (id) => del(`/emails/${id}`);
export const markEmailRead = (id) => post(`/emails/${id}/read`);
export const markEmailUnread = (id) => post(`/emails/${id}/unread`);
export const archiveEmail = (id) => post(`/emails/${id}/archive`);
export const untrashEmail = (id) => post(`/emails/${id}/untrash`);
export const fetchTrashEmails = (page = 1, pageSize = 50, filter = '') => 
  get('/emails/trash', { page: String(page), pageSize: String(pageSize), filter });
export const fetchArchivedEmails = (page = 1, pageSize = 50, filter = '') => 
  get('/emails/archived', { page: String(page), pageSize: String(pageSize), filter });
export const deleteEmailPermanently = (id) => del(`/emails/trash/${id}`);
export const unarchiveEmail = (id) => post(`/emails/${id}/unarchive`);

// Bulk action endpoints
export const bulkMarkRead = (emailIds) => post('/emails/bulk/read', { emailIds });
export const bulkMarkUnread = (emailIds) => post('/emails/bulk/unread', { emailIds });
export const bulkDelete = (emailIds) => post('/emails/bulk/delete', { emailIds });
export const bulkArchive = (emailIds) => post('/emails/bulk/archive', { emailIds });

// =============================================================================
// RULES ENDPOINTS
// =============================================================================

export const fetchRules = () => get('/rules');
export const createRule = (rule) => post('/rules', rule);
export const deleteRule = (id) => del(`/rules/${id}`);
export const updateRule = (id, rule) => put(`/rules/${id}`, rule);

// =============================================================================
// CLEANING ENDPOINTS
// =============================================================================

export const triggerClean = (data = null) => post('/clean', data);
export const previewClean = () => post('/clean/preview');
export const fetchCleanHistory = () => get('/clean/history');

// =============================================================================
// BLOCK & UNSUBSCRIBE ENDPOINTS
// =============================================================================

export const blockSender = (sender) => post('/block-sender', { sender });
export const unsubscribeFromNewsletter = (unsubscribeHeader, sender) => 
  post('/unsubscribe-newsletter', { unsubscribe_header: unsubscribeHeader, sender });

// =============================================================================
// IMAP ENDPOINTS
// =============================================================================

export const imapDelete = (host, username, password, uids) => 
  request('/imap/emails', {
    method: 'DELETE',
    body: JSON.stringify({ host, username, password, uids }),
  });

// =============================================================================
// ANALYTICS & STATISTICS ENDPOINTS
// =============================================================================

export const fetchSenderAnalytics = () => get('/analytics/top-senders');
export const fetchSubscribedSenders = () => get('/analytics/subscribed-senders');
export const fetchStats = () => get('/stats');

// =============================================================================
// SETTINGS ENDPOINTS
// =============================================================================

export const fetchSettings = () => get('/settings');
export const saveSettings = (settings) => post('/settings', settings);

// =============================================================================
// DEBUG/ADMIN ENDPOINTS
// =============================================================================

export const resetDatabase = () => post('/debug/reset-db');

// =============================================================================
// UTILITY FUNCTIONS
// =============================================================================

export const getErrorMessage = (error) => {
  if (!error) return 'An unknown error occurred';
  
  const message = error.message || String(error);
  
  // Map common error patterns to user-friendly messages
  if (message.includes('Network error') || message.includes('Failed to fetch')) {
    return 'Unable to connect to the server. Please check your internet connection.';
  }
  
  if (message.includes('timed out')) {
    return 'The request took too long. Please try again.';
  }
  
  if (message.includes('Session expired')) {
    return 'Your session has expired. Please sign in again.';
  }

  if (message.includes('Request failed with status: 403')) {
    return 'You do not have permission to perform this action.';
  }

  if (message.includes('Request failed with status: 404')) {
    return 'The requested resource could not be found.';
  }

  if (message.includes('Request failed with status: 500')) {
    return 'A server error occurred. Please try again later.';
  }

  return message;
};
