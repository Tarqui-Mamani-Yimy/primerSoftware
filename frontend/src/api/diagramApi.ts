import { UMLDiagramDocument } from '../types';
const runtimeApiBaseUrl = (globalThis as typeof globalThis & { __UML_API_BASE_URL__?: string }).__UML_API_BASE_URL__;
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || runtimeApiBaseUrl || 'http://localhost:8080/api/v1';
let accessToken: string | undefined;
export interface AuthenticatedUser { accessToken: string; userId: string; displayName: string; email: string; }
export interface AssignedProject { id: string; name: string; description: string; role: 'OWNER' | 'COLLABORATOR'; diagramCount: number; }
export interface DiagramSummary { id: string; name: string; updatedAt: string; }
export interface DiagramVersion { id: string; versionNumber: number; createdAt: string; document: UMLDiagramDocument; }
export class ApiError extends Error { constructor(public readonly status: number, message: string) { super(message); } }
async function request<T>(path: string, init?: RequestInit): Promise<T> { const response = await fetch(`${API_BASE_URL}${path}`, {...init,headers:{'Content-Type':'application/json',...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),...init?.headers}}); if(!response.ok){const body=await response.json().catch(()=>null) as {message?:string}|null; throw new ApiError(response.status,body?.message ?? `API request failed (${response.status})`);} return response.json() as Promise<T>; }
export const authApi={ login:(email:string,password:string)=>request<AuthenticatedUser>('/auth/login',{method:'POST',body:JSON.stringify({email,password})}), setToken:(token?:string)=>{accessToken=token;}, projects:()=>request<AssignedProject[]>('/projects') };
export const diagramApi={ list:(projectId:string)=>request<DiagramSummary[]>(`/projects/${projectId}/diagrams`), create:(projectId:string,d:UMLDiagramDocument)=>request<UMLDiagramDocument>(`/projects/${projectId}/diagrams`,{method:'POST',body:JSON.stringify(d)}), get:(p:string,id:string)=>request<UMLDiagramDocument>(`/projects/${p}/diagrams/${id}`), update:(p:string,id:string,d:UMLDiagramDocument)=>request<UMLDiagramDocument>(`/projects/${p}/diagrams/${id}`,{method:'PUT',body:JSON.stringify(d)}), versions:(p:string,id:string)=>request<DiagramVersion[]>(`/projects/${p}/diagrams/${id}/versions`), restore:(p:string,id:string,v:number)=>request<UMLDiagramDocument>(`/projects/${p}/diagrams/${id}/versions/${v}/restore`,{method:'POST'}) };
