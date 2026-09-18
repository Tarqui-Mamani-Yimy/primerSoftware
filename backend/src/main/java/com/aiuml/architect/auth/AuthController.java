package com.aiuml.architect.auth;
import jakarta.validation.Valid; import jakarta.validation.constraints.*; import org.springframework.web.bind.annotation.*;
@RestController @RequestMapping("/api/v1/auth") class AuthController { private final AuthService service; AuthController(AuthService s){service=s;} @PostMapping("/login") AuthService.LoginResponse login(@Valid @RequestBody LoginInput input){return service.login(new AuthService.LoginRequest(input.email(),input.password()));} record LoginInput(@NotBlank @Email String email,@NotBlank String password){} }
