import axios from 'axios'

const client = axios.create({
    baseURL: '/api',
    timeout: 30000,
})

// Inject Authorization header
client.interceptors.request.use((config) => {
    const token = localStorage.getItem('token')
    if (token) {
        config.headers.Authorization = `Bearer ${token}`
    }
    return config
})

// Handle 401 → clear token and redirect to login
client.interceptors.response.use(
    (response) => response,
    (error) => {
        if (error.response?.status === 401) {
            localStorage.removeItem('token')
            // Only redirect if not already on a public page
            if (!window.location.pathname.startsWith('/login') && window.location.pathname !== '/') {
                window.location.href = '/'
            }
        }
        return Promise.reject(error)
    },
)

export default client
