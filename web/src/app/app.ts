import { Component, inject } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { SessionService } from './session';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet],
  template: `
    <header class="navbar bg-body-tertiary border-bottom mb-4">
      <div class="container">
        <span class="navbar-brand mb-0 h1">Flashcards</span>
        @if (session.session(); as s) {
          <span class="navbar-text small">
            {{ s.name || 'Signed in' }}
            <span class="badge ms-1" [class.text-bg-primary]="s.instructor" [class.text-bg-secondary]="!s.instructor">
              {{ s.instructor ? 'Instructor' : 'Student' }}
            </span>
          </span>
        }
      </div>
    </header>

    <main class="container pb-5">
      @if (session.token()) {
        <router-outlet />
      } @else {
        <div class="alert alert-info" role="alert">
          @if (session.expired()) {
            Your session has expired.
          }
          Open Flashcards from your course in your learning management system to get started.
        </div>
      }
    </main>
  `,
})
export class App {
  protected readonly session = inject(SessionService);
}
