import { Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet],
  template: `
    <header class="navbar bg-body-tertiary border-bottom mb-4">
      <div class="container">
        <span class="navbar-brand mb-0 h1">Flashcards</span>
      </div>
    </header>

    <main class="container pb-5">
      <router-outlet />
    </main>
  `,
})
export class App {}
