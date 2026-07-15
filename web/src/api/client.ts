import axios from 'axios'

const client = axios.create({
    baseURL: '/api',
    timeout: 30000,
    withCredentials: true,
})

// Session is HttpOnly cookie based — no Authorization header from localStorage.
client.interceptors.response.use(
    (response) => response,
    (error) => {
        if (error.response?.status === 401) {
            // Only redirect when not already on public pages
            if (!window.location.pathname.startsWith('/login') && window.location.pathname !== '/' && !window.location.pathname.startsWith('/share/')) {
                window.location.href = '/'
            }
        }
        return Promise.reject(error)
    },
)

export default client
