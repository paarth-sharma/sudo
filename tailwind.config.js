/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./templates/**/*.{templ,html}", "./static/**/*.js"],
  theme: {
    extend: {
      colors: {
        'priority-low': '#10B981',
        'priority-medium': '#F59E0B',
        'priority-high': '#EF4444',
        'priority-urgent': '#DC2626',
      }
    },
  },
  plugins: [],
}