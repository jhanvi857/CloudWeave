'use client'

import React, { useRef, useState, useMemo, useEffect } from 'react'
import { Canvas, useFrame, ThreeEvent } from '@react-three/fiber'
import { OrbitControls, Html } from '@react-three/drei'
import * as THREE from 'three'

interface FullBleedVaultLock3DProps {
  isLocked: boolean
  setIsLocked: (locked: boolean | ((prev: boolean) => boolean)) => void
}

// Radial Dial Ticks on the Vault Lock
function LockTicks({ count = 72, radius = 2.4 }: { count?: number; radius?: number }) {
  const lines = useMemo(() => {
    const points: THREE.Vector3[][] = []
    for (let i = 0; i < count; i++) {
      const angle = (i * 2 * Math.PI) / count
      const isMajor = i % 12 === 0
      const isSemi = i % 4 === 0
      const innerR = radius - (isMajor ? 0.3 : isSemi ? 0.18 : 0.08)
      const outerR = radius + (isMajor ? 0.12 : 0.04)

      const x1 = Math.cos(angle) * innerR
      const y1 = Math.sin(angle) * innerR
      const x2 = Math.cos(angle) * outerR
      const y2 = Math.sin(angle) * outerR

      points.push([new THREE.Vector3(x1, y1, 0), new THREE.Vector3(x2, y2, 0)])
    }
    return points
  }, [count, radius])

  return (
    <group>
      {lines.map((line, idx) => {
        const isMajor = idx % 12 === 0
        const isSemi = idx % 4 === 0
        const color = isMajor ? '#7A1F2B' : isSemi ? '#A6A9AE' : '#26262B'
        const geom = new THREE.BufferGeometry().setFromPoints(line)
        return (
          <primitive
            key={idx}
            object={
              new THREE.Line(
                geom,
                new THREE.LineBasicMaterial({
                  color,
                  transparent: true,
                  opacity: isMajor ? 0.95 : 0.4,
                })
              )
            }
          />
        )
      })}
    </group>
  )
}

function VaultLockMechanismScene({
  isLocked,
  setIsLocked,
}: {
  isLocked: boolean
  setIsLocked: (locked: boolean | ((prev: boolean) => boolean)) => void
}) {
  const groupRef = useRef<THREE.Group>(null)
  const tumblerRef = useRef<THREE.Group>(null)
  const bolt1Ref = useRef<THREE.Mesh>(null)
  const bolt2Ref = useRef<THREE.Mesh>(null)
  const flashRingRef = useRef<THREE.Mesh>(null)

  const [hovered, setHovered] = useState(false)
  const flashIntensity = useRef(0)
  const currentAngle = useRef(0)

  useEffect(() => {
    document.body.style.cursor = hovered ? 'pointer' : 'auto'
    return () => {
      document.body.style.cursor = 'auto'
    }
  }, [hovered])

  // Trigger flash on lock change
  useEffect(() => {
    if (isLocked) {
      flashIntensity.current = 3.5
    }
  }, [isLocked])

  useFrame((_, delta) => {
    // Idle rotation when unlocked
    if (!isLocked && groupRef.current) {
      groupRef.current.rotation.z += delta * 0.08
    }

    // Spring interpolation toward target angle (0 when unlocked, PI / 2 when locked)
    const targetAngle = isLocked ? Math.PI / 2 : 0
    currentAngle.current = THREE.MathUtils.damp(currentAngle.current, targetAngle, 14, delta)

    if (tumblerRef.current) {
      tumblerRef.current.rotation.z = currentAngle.current
    }

    // Locking bolts extension/retraction
    const targetBoltPos = isLocked ? 2.1 : 1.3
    if (bolt1Ref.current && bolt2Ref.current) {
      bolt1Ref.current.position.y = THREE.MathUtils.damp(
        bolt1Ref.current.position.y,
        targetBoltPos,
        18,
        delta
      )
      bolt2Ref.current.position.y = THREE.MathUtils.damp(
        bolt2Ref.current.position.y,
        -targetBoltPos,
        18,
        delta
      )
    }

    // Decay flash intensity
    if (flashRingRef.current) {
      flashIntensity.current = THREE.MathUtils.damp(flashIntensity.current, 0, 4, delta)
      const mat = flashRingRef.current.material as THREE.MeshStandardMaterial
      if (mat) {
        mat.emissiveIntensity = flashIntensity.current
        mat.opacity = Math.min(1, flashIntensity.current * 0.6)
      }
    }
  })

  const handleClick = (e: ThreeEvent<MouseEvent>) => {
    e.stopPropagation()
    setIsLocked((prev) => !prev)
  }

  return (
    <group
      ref={groupRef}
      onClick={handleClick}
      onPointerOver={() => setHovered(true)}
      onPointerOut={() => setHovered(false)}
    >
      <ambientLight intensity={0.6} />
      <directionalLight position={[6, 8, 6]} intensity={1.6} color="#EDE8DF" />
      <directionalLight position={[-6, -6, 4]} intensity={0.9} color="#7A1F2B" />
      <pointLight position={[0, 0, 2]} intensity={isLocked ? 2.5 : 1.2} color="#7A1F2B" distance={8} />

      {/* Base Outer Bezel Torus */}
      <mesh position={[0, 0, -0.1]}>
        <torusGeometry args={[2.7, 0.16, 24, 64]} />
        <meshStandardMaterial
          color="#A6A9AE"
          metalness={0.92}
          roughness={0.15}
        />
      </mesh>

      {/* Outer Chamber Base Cylinder */}
      <mesh position={[0, 0, -0.2]}>
        <cylinderGeometry args={[2.68, 2.72, 0.25, 64]} />
        <meshStandardMaterial
          color="#19191C"
          metalness={0.88}
          roughness={0.3}
        />
      </mesh>

      {/* Inner Recessed Ring */}
      <mesh position={[0, 0, 0]}>
        <torusGeometry args={[2.4, 0.05, 16, 64]} />
        <meshStandardMaterial
          color="#26262B"
          metalness={0.8}
          roughness={0.4}
        />
      </mesh>

      {/* Radial Ticks */}
      <LockTicks count={72} radius={2.4} />

      {/* Flash Ring Pulse on Lock */}
      <mesh ref={flashRingRef} position={[0, 0, 0.02]}>
        <torusGeometry args={[2.4, 0.08, 16, 64]} />
        <meshStandardMaterial
          color="#EDE8DF"
          emissive="#7A1F2B"
          emissiveIntensity={0}
          transparent
          opacity={0}
        />
      </mesh>

      {/* 4 Perimeter Locking Notches */}
      {[0, Math.PI / 2, Math.PI, (3 * Math.PI) / 2].map((angle, i) => (
        <group key={i} rotation={[0, 0, angle]}>
          <mesh position={[2.42, 0, 0]}>
            <boxGeometry args={[0.22, 0.18, 0.2]} />
            <meshStandardMaterial color="#7A1F2B" metalness={0.9} roughness={0.2} />
          </mesh>
        </group>
      ))}

      {/* Central Rotating Tumbler Hub & Crossbar */}
      <group ref={tumblerRef} position={[0, 0, 0.05]}>
        {/* Central Spindle Hub Cylinder */}
        <mesh>
          <cylinderGeometry args={[0.75, 0.85, 0.35, 32]} />
          <meshStandardMaterial
            color="#19191C"
            metalness={0.92}
            roughness={0.2}
            emissive={isLocked ? '#421118' : '#000000'}
            emissiveIntensity={isLocked ? 0.6 : 0.1}
          />
        </mesh>

        {/* Central Embossed Garnet Plate */}
        <mesh position={[0, 0, 0.19]}>
          <cylinderGeometry args={[0.42, 0.42, 0.08, 24]} />
          <meshStandardMaterial
            color="#7A1F2B"
            metalness={0.95}
            roughness={0.1}
            emissive="#7A1F2B"
            emissiveIntensity={isLocked ? 0.8 : 0.3}
          />
        </mesh>

        {/* Central Steel Ring */}
        <mesh position={[0, 0, 0.22]}>
          <torusGeometry args={[0.38, 0.03, 16, 32]} />
          <meshStandardMaterial color="#EDE8DF" metalness={0.95} roughness={0.1} />
        </mesh>

        {/* Tumbler Crossbar Beam */}
        <mesh position={[0, 0, 0.12]}>
          <boxGeometry args={[0.24, 2.6, 0.18]} />
          <meshStandardMaterial
            color={isLocked ? '#7A1F2B' : '#A6A9AE'}
            metalness={0.9}
            roughness={0.15}
            emissive={isLocked ? '#421118' : '#000000'}
            emissiveIntensity={isLocked ? 0.6 : 0.1}
          />
        </mesh>

        {/* Interlocking Locking Bolts */}
        <mesh ref={bolt1Ref} position={[0, 1.3, 0.12]}>
          <cylinderGeometry args={[0.1, 0.1, 0.6, 16]} />
          <meshStandardMaterial
            color="#EDE8DF"
            metalness={0.95}
            roughness={0.1}
            emissive="#7A1F2B"
            emissiveIntensity={isLocked ? 0.9 : 0.2}
          />
        </mesh>

        <mesh ref={bolt2Ref} position={[0, -1.3, 0.12]}>
          <cylinderGeometry args={[0.1, 0.1, 0.6, 16]} />
          <meshStandardMaterial
            color="#EDE8DF"
            metalness={0.95}
            roughness={0.1}
            emissive="#7A1F2B"
            emissiveIntensity={isLocked ? 0.9 : 0.2}
          />
        </mesh>

        {/* Interactive Indicator HUD on Dial */}
        <Html position={[0, 0, 0.35]} center distanceFactor={8}>
          <div
            className={`px-3 py-1 rounded-full font-mono text-[10px] whitespace-nowrap border shadow-xl transition-all cursor-pointer select-none ${isLocked
                ? 'bg-[#7A1F2B] text-[#EDE8DF] font-bold border-[#EDE8DF] ring-2 ring-[#EDE8DF]/40'
                : hovered
                  ? 'bg-[#19191C] text-[#EDE8DF] border-[#7A1F2B] scale-105'
                  : 'bg-[#000000]/90 text-[#A6A9AE] border-[#26262B]'
              }`}
          >
            {isLocked ? '✓ CLUSTER LOCKED (SEALED)' : hovered ? 'CLICK TO ENGAGE LOCK' : 'RING: UNLOCKED'}
          </div>
        </Html>
      </group>
    </group>
  )
}

export function FullBleedVaultLock3D({
  isLocked,
  setIsLocked,
}: FullBleedVaultLock3DProps) {
  const [prefersReducedMotion, setPrefersReducedMotion] = useState(false)

  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
    setPrefersReducedMotion(mediaQuery.matches)
    const handler = (e: MediaQueryListEvent) => setPrefersReducedMotion(e.matches)
    mediaQuery.addEventListener('change', handler)
    return () => mediaQuery.removeEventListener('change', handler)
  }, [])

  if (prefersReducedMotion) {
    return (
      <div className="w-full h-full flex flex-col items-center justify-center p-6 text-center">
        <button
          onClick={() => setIsLocked((prev) => !prev)}
          className={`px-6 py-3 rounded-xl border font-mono text-sm transition-all cursor-pointer ${isLocked
              ? 'bg-[#7A1F2B] text-[#EDE8DF] border-[#EDE8DF] font-bold'
              : 'bg-[#19191C] text-[#A6A9AE] border-[#26262B]'
            }`}
        >
          {isLocked ? '✓ Cluster Sealed & Locked' : 'Click to Engage Cluster Lock'}
        </button>
      </div>
    )
  }

  return (
    <div className="w-full h-full cursor-pointer select-none">
      <Canvas
        camera={{ position: [0, 0, 7.5], fov: 42 }}
        gl={{ antialias: true, alpha: true }}
        className="w-full h-full"
      >
        <VaultLockMechanismScene isLocked={isLocked} setIsLocked={setIsLocked} />
        <OrbitControls
          enablePan={false}
          enableZoom={false}
          maxPolarAngle={Math.PI / 2 + 0.15}
          minPolarAngle={Math.PI / 2 - 0.15}
          maxAzimuthAngle={0.2}
          minAzimuthAngle={-0.2}
        />
      </Canvas>
    </div>
  )
}
