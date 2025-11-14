Option 1: Using Air (Recommended for Development)

  Simply restart Air:

  1. In the terminal where Air is running, press Ctrl+C to stop it
  2. Start Air again:
  air
  3. Air will automatically:
    - Run templ generate
    - Build Tailwind CSS
    - Build the Go binary to tmp/main.exe
    - Start the server
  4. Watch for this log line to confirm the schema fix is loaded:
  Supabase client initialized with URL: https://..., using public schema
  5. Now try deleting tasks or your account - it should work!

  Option 2: Run the Binary Directly

  If you want to run the pre-built binary instead:

  1. Stop Air (Ctrl+C)
  2. Build manually (if needed):
  templ generate
  npx tailwindcss -i ./static/css/input.css -o ./static/css/styles.css
  go build -o bin/server.exe ./cmd/server
  3. Run the binary:
  ./bin/server.exe

  Quick Fix Right Now:

  Just restart Air:
  # Press Ctrl+C to stop Air, then:
  air

  That's it! Air will rebuild everything with the schema configuration fix and the        
  account deletion feature will work. Air is definitely the better choice for local       
  development since it gives you hot reloading. The schema configuration will be
  included in the new build.