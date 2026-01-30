import axios from 'axios'

const client = axios.create({
  baseURL: (import.meta as any).env?.PROD ? '' : '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

client.interceptors.response.use(
  (response) => response,
  (error) => {
    console.error('API Error:', error)
    return Promise.reject(error)
  }
)

export default client
